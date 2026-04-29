# # Simple UDP DDoS Tool

Termux və Python ilə işləyən sadə UDP Flood tool.

**Xəbərdarlıq:** Bu tool yalnız təhsil və test məqsədilə istifadə olunmalıdır. Qanunsuz hücum etmək cinayətdir!

### Termux-da Qurulum

```bash
pkg update && pkg upgrade -y
pkg install python git -y
pip install --upgrade pip

git clone https://github.com/seninusername/ddos-tool.git
cd ddos-tool
python ddos.py <HƏDƏF_IP> <PORT> [THREAD_SAYI]

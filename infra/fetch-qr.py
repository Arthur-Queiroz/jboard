#!/usr/bin/env python3
"""Busca QR code da Evolution API e salva na pasta do projeto.
Usa urllib (não curl) pra evitar bug de quoting do zsh."""
import json
import urllib.request
import base64
import os
import time

EVO_URL = "http://localhost:8081"
INSTANCE = "jboard"
API_KEY = "changeme"
OUTPUT = os.path.expanduser("~/prog/jboard/qrcode.png")

def api(method, path, body=None):
    data = json.dumps(body).encode() if body else None
    req = urllib.request.Request(
        f"{EVO_URL}{path}",
        data=data,
        method=method,
        headers={"apikey": API_KEY, "Content-Type": "application/json"},
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return resp.status, json.loads(resp.read())
    except urllib.error.HTTPError as e:
        try:
            return e.code, json.loads(e.read())
        except Exception:
            return e.code, {"error": str(e)}
    except Exception as e:
        return 0, {"error": str(e)}

print(f"Buscando QR code da instância '{INSTANCE}'...")
for attempt in range(10):
    status, data = api("GET", f"/instance/connect/{INSTANCE}")
    print(f"  Tentativa {attempt+1}: status={status}")

    if status == 200:
        qr_b64 = None
        if isinstance(data, dict):
            qr_b64 = data.get("base64")
            if not qr_b64 and "qrcode" in data:
                qr_b64 = data["qrcode"].get("base64")

        if qr_b64:
            if "," in qr_b64:
                qr_b64 = qr_b64.split(",", 1)[1]
            png_data = base64.b64decode(qr_b64)
            with open(OUTPUT, "wb") as f:
                f.write(png_data)
            print(f"\n>>> QR code salvo em {OUTPUT}")
            print(f">>> Tamanho: {len(png_data)} bytes")
            print(f"\nABRA AGORA e escaneie em até 30 segundos:")
            print(f"  {OUTPUT}")
            print(f"\nWhatsApp → Configurações → Aparelhos conectados → Conectar aparelho")
            exit(0)
        else:
            print(f"  Sem base64. Campos: {list(data.keys()) if isinstance(data, dict) else type(data)}")

    time.sleep(3)

print("\nFalha. Tente deletar e recriar a instância.")

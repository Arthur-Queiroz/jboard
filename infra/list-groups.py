#!/usr/bin/env python3
"""Lista grupos do WhatsApp via Evolution API."""
import json
import urllib.request

EVO_URL = "http://localhost:8081"
INSTANCE = "jboard"
API_KEY = "changeme"

def api(method, path):
    req = urllib.request.Request(
        f"{EVO_URL}{path}",
        method=method,
        headers={"apikey": API_KEY, "Content-Type": "application/json"},
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return resp.status, json.loads(resp.read())
    except Exception as e:
        return 0, {"error": str(e)}

print("Buscando grupos...")
status, data = api("GET", f"/group/fetchAll/{INSTANCE}")
print(f"Status: {status}")
print(json.dumps(data, indent=2, ensure_ascii=False))

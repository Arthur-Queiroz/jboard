#!/usr/bin/env python3
"""Teste do fluxo completo do jboard: board → column → card → reminder.
Cria um lembrete pra daqui a 2 minutos e o scheduler dispara via WhatsApp."""
import json
import urllib.request
from datetime import datetime, timedelta, timezone

API = "http://localhost:8080/api"

def post(path, body):
    req = urllib.request.Request(
        f"{API}{path}",
        data=json.dumps(body).encode(),
        method="POST",
        headers={"Content-Type": "application/json"},
    )
    with urllib.request.urlopen(req, timeout=10) as resp:
        return resp.status, json.loads(resp.read())

def get(path):
    req = urllib.request.Request(f"{API}{path}", method="GET")
    with urllib.request.urlopen(req, timeout=10) as resp:
        return resp.status, json.loads(resp.read())

# 1. Criar board
_, board = post("/boards", {"title": "Teste Scheduler"})
print(f"Board criado: id={board['id']}, title={board['title']}")

# 2. Criar coluna
_, column = post(f"/boards/{board['id']}/columns", {"title": "A fazer", "position": 0})
print(f"Column criada: id={column['id']}, title={column['title']}")

# 3. Criar card
_, card = post(f"/columns/{column['id']}/cards", {"title": "Lembrete teste", "position": 0})
print(f"Card criado: id={card['id']}, title={card['title']}")

# 4. Criar lembrete pra daqui a 2 minutos
reminder_at = (datetime.now(timezone.utc) + timedelta(minutes=2)).isoformat()
_, reminder = post(f"/cards/{card['id']}/reminders", {
    "reminder_at": reminder_at,
    "message": "⏰ Lembrete do jboard: fluxo completo funcionando!",
})
print(f"Reminder criado: id={reminder['id']}, reminder_at={reminder['reminder_at']}")
print(f"  recipient={reminder['recipient']}")
print(f"  message={reminder['message']}")

print(f"\n>>> Aguarde ~2 minutos. O scheduler vai disparar a mensagem no grupo do WhatsApp.")
print(f">>> Hora atual: {datetime.now(timezone.utc).isoformat()}")
print(f">>> Hora do lembrete: {reminder_at}")

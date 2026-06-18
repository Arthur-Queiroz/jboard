#!/usr/bin/env bash
# setup-evolution.sh — cria instância WhatsApp na Evolution API e mostra QR code.
#
# Uso:
#   cd jboard/infra
#   cp .env.example .env  # editar .env com API key + número
#   docker compose up -d
#   sleep 15              # aguardar Evolution API subir
#   ./setup-evolution.sh
set -euo pipefail

INSTANCE="${JBOARD_EVOLUTION_INSTANCE:-jboard}"
API_KEY="${JBOARD_EVOLUTION_API_KEY:-changeme}"
EVO_URL="http://localhost:8081"
QR_FILE="/tmp/jboard-qrcode.png"

echo "=== jboard — Evolution API setup ==="
echo "Instância: $INSTANCE"
echo "URL: $EVO_URL"
echo "API Key: $API_KEY"
echo ""

# Espera a Evolution API ficar pronta.
echo "Aguardando Evolution API responder..."
for i in $(seq 1 30); do
  if curl -sf "$EVO_URL" >/dev/null 2>&1; then
    echo "Evolution API no ar."
    break
  fi
  sleep 1
  [ "$i" -eq 30 ] && { echo "Timeout: Evolution API não respondeu em 30s."; exit 1; }
done

# Helper: extrai base64 do QR de uma string JSON.
extract_qr() {
  python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    # v2: QR pode estar em data['qrcode']['base64'] ou data['base64']
    qr = ''
    if 'qrcode' in data and isinstance(data['qrcode'], dict):
        qr = data['qrcode'].get('base64', '')
    elif 'base64' in data:
        qr = data['base64', '']
    if ',' in qr:
        qr = qr.split(',', 1)[1]
    print(qr)
except Exception:
    print('')
"
}

echo ""
echo ">>> Criando instância \"$INSTANCE\"..."

# Cria com payload de arquivo pra evitar problemas de quoting do shell.
PAYLOAD_FILE=$(mktemp)
cat > "$PAYLOAD_FILE" <<EOF
{"instanceName": "$INSTANCE", "integration": "WHATSAPP-BAILEYS", "qrcode": true}
EOF

CREATE_BODY=$(curl -s -w "\n%{http_code}" \
  -X POST "$EVO_URL/instance/create" \
  -H "Content-Type: application/json" \
  -H "apikey: $API_KEY" \
  -d @"$PAYLOAD_FILE" 2>&1)
rm -f "$PAYLOAD_FILE"

HTTP_CODE=$(echo "$CREATE_BODY" | tail -1)
BODY=$(echo "$CREATE_BODY" | sed '$d')

echo "HTTP: $HTTP_CODE"
echo "Resposta: $BODY" | cut -c1-200
echo ""

# Tenta extrair QR da resposta do create.
QR_BASE64=$(echo "$BODY" | extract_qr)

if [ -n "$QR_BASE64" ]; then
  echo ">>> QR code encontrado na resposta do create!"
else
  echo ">>> QR não veio no create. Tentando /instance/connect (poll 5x, 3s cada)..."

  for i in 1 2 3 4 5; do
    sleep 3
    CONNECT_BODY=$(curl -s \
      -X GET "$EVO_URL/instance/connect/$INSTANCE" \
      -H "apikey: $API_KEY" 2>&1)

    echo "  Tentativa $i: $(echo "$CONNECT_BODY" | cut -c1-100)"

    QR_BASE64=$(echo "$CONNECT_BODY" | extract_qr)
    if [ -n "$QR_BASE64" ]; then
      echo "  QR code encontrado!"
      break
    fi
  done
fi

if [ -z "$QR_BASE64" ]; then
  echo ""
  echo ">>> Não foi possível obter o QR code automaticamente."
  echo ""
  echo "Tente manualmente — cole isto no navegador como URL:"
  echo "  data:image/png;base64,COLE_O_BASE64_AQUI"
  echo ""
  echo "Ou rode e procure pelo campo 'base64' na resposta:"
  echo "  curl -s http://localhost:8081/instance/connect/$INSTANCE -H 'apikey: $API_KEY' | python3 -m json.tool"
  exit 1
fi

# Salva o QR code como PNG.
echo "$QR_BASE64" | base64 -d > "$QR_FILE"
echo ""
echo "============================================"
echo ">>> QR code salvo em $QR_FILE"
echo "============================================"
echo ""
echo "PRÓXIMOS PASSOS:"
echo "  1. Abra $QR_FILE no visualizador de imagens"
echo "     (ou cole no navegador: data:image/png;base64,$QR_BASE64)"
echo "  2. No celular: WhatsApp → Configurações → Aparelhos conectados → Conectar aparelho"
echo "  3. Escaneie o QR code"
echo "  4. Aguarde — o status muda pra 'open'"
echo ""
echo "Verificar status:"
echo "  curl -s http://localhost:8081/instance/connect/$INSTANCE -H 'apikey: $API_KEY' | python3 -m json.tool"
echo ""
echo "Mensagem de teste (substitua o número):"
echo "  curl -s -X POST http://localhost:8081/message/sendText/$INSTANCE \\"
echo "    -H 'Content-Type: application/json' -H 'apikey: $API_KEY' \\"
echo "    -d '{\"number\": \"5511999999999\", \"text\": \"jboard!\"}'"

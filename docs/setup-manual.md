# jboard — Guia de Setup Manual

Tudo que exige ação manual sua, por fase. Cada seção é independente — faça na
ordem que preferir.

---

## 1. Dependências do Tauri (Fase 4 — desktop)

O `cargo build` em `desktop/src-tauri/` precisa das libs GTK/webkit do Linux.
Instalar uma única vez no WSL:

```bash
sudo apt-get update && sudo apt-get install -y \
  libwebkit2gtk-4.1-dev build-essential \
  curl wget file libxdo-dev libssl-dev \
  libayatana-appindicator3-dev librsvg2-dev pkg-config
```

Depois disso, validar:

```bash
cd desktop/src-tauri
cargo check    # valida código + linkagem (~2min na primeira vez)
cargo build    # build de desenvolvimento
```

Pra gerar o binário release + instalador:

```bash
cd frontend
npm run tauri:build    # build Vue → compila Rust → empacota (.deb/.AppImage)
```

Pra abrir em modo dev (janela + hot reload do Vue):

```bash
cd frontend
npm run tauri:dev
```

---

## 2. Evolution API + WhatsApp (Fase 2 — lembretes)

### 2.1. Visão geral

A Evolution API v2 roda como container Docker e fala com o WhatsApp via Baileys
(WhatsApp Web). Precisa de Postgres (Prisma ORM) + Redis. O `docker-compose.yml`
já inclui tudo: Postgres compartilhado (databases `jboard` + `evolution`),
Redis dedicado, e a Evolution API na porta **8081** (só pra setup — em produção
remove o mapeamento).

O fluxo é:
1. Subir os containers
2. Criar uma "instância" WhatsApp na Evolution API
3. Escanear o QR code com o celular
4. Configurar o número destinatário no `.env`
5. Testar o envio

### 2.2. Configurar o `.env`

```bash
cd infra
cp .env.example .env
```

Editar `.env`:

```bash
# Escolha uma API key (pode ser qualquer string, mas use algo aleatório em produção)
JBOARD_EVOLUTION_API_KEY=<gere com: openssl rand -hex 16>

# Nome da instância (pode deixar "jboard")
JBOARD_EVOLUTION_INSTANCE=jboard

# Seu número de WhatsApp com DDI+DDD, sem + e sem espaços
# Ex: Brasil (55) + São Paulo (11) + número = 5511999999999
JBOARD_WHATSAPP_RECIPIENT=5511999999999
```

### 2.3. Subir os containers

```bash
cd infra
docker compose up -d
```

Verificar se tudo subiu:

```bash
docker compose ps
# Deve mostrar: postgres (healthy), redis (up), evolution-api (up), backend (up)
```

Se a Evolution API não estiver `up`, checar logs:

```bash
docker compose logs evolution-api --tail 30
```

### 2.4. Criar a instância e escanear o QR code

**Opção A — Script automatizado** (recomendado):

```bash
./infra/setup-evolution.sh
```

O script:
- Cria a instância `jboard` na Evolution API
- Busca o QR code e salva em `/tmp/jboard-qrcode.png`
- Imprime instruções de como escanear

**Opção B — Manual** (se o script falhar):

```bash
# 1. Criar instância (já com QR code na resposta)
curl -s -X POST http://localhost:8081/instance/create \
  -H "Content-Type: application/json" \
  -H "apikey: $(grep JBOARD_EVOLUTION_API_KEY .env | cut -d= -f2)" \
  -d '{"instanceName": "jboard", "qrcode": true}' | python3 -m json.tool

# A resposta inclui "base64": "data:image/png;base64,iVBORw0KGgo..."
# Copie o base64 (sem o prefixo data:image/png;base64,) e salve como PNG,
# ou cole a string completa data:image/png;base64,... na barra de URL do navegador.

# 2. Se precisar de um QR code novo (expiram em ~60s):
curl -s http://localhost:8081/instance/connect/jboard \
  -H "apikey: $(grep JBOARD_EVOLUTION_API_KEY .env | cut -d= -f2)" | python3 -m json.tool
```

### 2.5. Escanear com o celular

1. Abra o QR code (`/tmp/jboard-qrcode.png` ou cole o base64 no navegador)
2. No celular: **WhatsApp → Configurações → Aparelhos conectados → Conectar aparelho**
3. Escaneie o QR code
4. Aguarde 10-20s — o WhatsApp conecta e a instância fica "open"

Verificar o status da conexão:

```bash
curl -s http://localhost:8081/instance/connect/jboard \
  -H "apikey: $(grep JBOARD_EVOLUTION_API_KEY .env | cut -d= -f2)" | python3 -m json.tool
# "status": "open" = conectado
```

### 2.6. Testar o envio de mensagem

Substitua `SEU_NUMERO` pelo número que vai receber a mensagem (com DDI+DDD):

```bash
API_KEY=$(grep JBOARD_EVOLUTION_API_KEY infra/.env | cut -d= -f2)

curl -s -X POST http://localhost:8081/message/sendText/jboard \
  -H "Content-Type: application/json" \
  -H "apikey: $API_KEY" \
  -d '{"number": "5511999999999", "text": "jboard funcionando!"}' | python3 -m json.tool
```

Se a mensagem chegou no WhatsApp, a Evolution API está pronta.

### 2.7. Testar o fluxo completo (lembrete do jboard)

O backend Go já está configurado pra usar a Evolution API. Com os containers no ar:

```bash
# 1. Subir o backend (se não estiver rodando via docker compose)
cd backend
JBOARD_DB_PASSWORD=jboard \
JBOARD_EVOLUTION_URL=http://localhost:8081 \
JBOARD_EVOLUTION_INSTANCE=jboard \
JBOARD_EVOLUTION_API_KEY=$(grep JBOARD_EVOLUTION_API_KEY ../infra/.env | cut -d= -f2) \
JBOARD_WHATSAPP_RECIPIENT=5511999999999 \
/usr/local/go/bin/go run ./cmd/server

# 2. Em outro terminal: criar um lembrete pra daqui a 2 minutos
#    (substitua o datetime pelo horário atual + 2 min, em ISO 8601)
curl -s -X POST http://localhost:8080/api/boards -H "Content-Type: application/json" -d '{"title":"Teste"}'
curl -s -X POST http://localhost:8080/api/boards/1/columns -H "Content-Type: application/json" -d '{"title":"A fazer","position":0}'
curl -s -X POST http://localhost:8080/api/columns/1/cards -H "Content-Type: application/json" -d '{"title":"Lembrete teste","position":0}'

# Lembrete pra daqui a 2 minutos (ajuste o horário):
curl -s -X POST http://localhost:8080/api/cards/1/reminders \
  -H "Content-Type: application/json" \
  -d '{"reminder_at": "2026-06-17T22:30:00Z", "message": "Lembrete do jboard!"}'

# 3. Aguardar 2 minutos — o scheduler dispara e a mensagem chega no WhatsApp
#    Log no terminal do backend: "scheduler: enviar lembrete 1: ..."
```

### 2.8. Considerações sobre ban

Uso pessoal, single-recipient, com intervalos de horas entre mensagens = perfil
de baixo risco. Os padrões de detecção do WhatsApp giram em torno de spam/bulk,
não desse uso. Se quiser eliminar qualquer risco residual pro número principal,
use um número secundário pra automação (qualquer chip pré-pago serve).

---

## 3. Rodar o sistema completo localmente

```bash
# 1. Subir infra (Postgres + Redis + Evolution API + backend)
cd infra
cp .env.example .env  # editar .env com API key + número
docker compose up -d

# 2. Setup da Evolution API (só na primeira vez)
./setup-evolution.sh

# 3. Frontend dev (hot reload)
cd ../frontend
npm install
npm run dev    # http://localhost:5173

# 4. (Opcional) Desktop Tauri
#    Precisa das libs do passo 1 instaladas
npm run tauri:dev
```

---

## 4. Deploy produção

O deploy de produção atual usa Docker Compose + Caddy + Cloudflare Tunnel na VPS.
O runbook canônico fica em `docs/deploy.md`.

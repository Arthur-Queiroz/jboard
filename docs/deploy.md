# jboard — Deploy na VPS (manual)

Deploy de produção na Hostinger KVM2 atrás do Cloudflare Tunnel, método
**Docker Compose + git pull** (ver `dev-notes/playbook-cicd-compose-cloudflare-tunnel`
e `dev-notes/deploy-kvm2-hostinger-cloudflared` no Obsidian). Este guia é o
deploy **manual**; o CI/CD via GitHub Actions vem por cima depois.

## Arquitetura

```
Internet → Cloudflare Edge → Tunnel (cloudflared) → 127.0.0.1:8083 → Caddy (web)
                                                                       ├─ /api/* → backend:8080 (Go)
                                                                       └─ /*     → SPA estático (Vue dist)
backend → postgres (dedicado) · evolution-api → postgres(db evolution) + redis
```

**Subdomínio único** `jboard.devarthur.com.br`: o Caddy serve o SPA e faz
`reverse_proxy` de `/api` pro backend na **mesma origem** — sem CORS. Só o
serviço `web` abre porta, e no loopback (`127.0.0.1:8083`); Postgres, Redis,
backend e Evolution só existem na rede interna do compose.

> Mapa de portas da VPS: 5678 n8n · 8080 Evolution(antiga) · 8081 jpad ·
> 8082/8443 jblog · **8083 jboard** · 8084 jboard-evolution (só setup).

## Pré-requisitos na VPS (uma vez)

A VPS já tem Docker + cloudflared (do jpad/jblog). Falta só a rota do tunnel
pro jboard. **A rota SSH e o sshd hardening já existem** (jpad/jblog) — não
mexer.

## Passo a passo

### 1. Clonar o repo na VPS

Repo precisa estar no GitHub (público, ou com deploy key se privado). Na VPS:

```bash
ssh root@2.25.158.85
git clone https://github.com/Arthur-Queiroz/jboard /opt/jboard
cd /opt/jboard
```

### 2. Criar o `infra/.env` de produção

```bash
cd /opt/jboard/infra
cp .env.example .env
```

Editar `.env` com valores **reais** (todos obrigatórios em prod):

```bash
JBOARD_DB_PASSWORD=$(openssl rand -hex 24)       # cole o valor gerado
JBOARD_EVOLUTION_API_KEY=$(openssl rand -hex 16) # cole o valor gerado
JBOARD_EVOLUTION_INSTANCE=jboard
JBOARD_WHATSAPP_RECIPIENT=55XXXXXXXXXXX           # ou o @g.us do grupo
JBOARD_API_TOKEN=$(openssl rand -hex 32)          # cole o valor gerado
```

> O `compose.prod` **falha de propósito** se `JBOARD_DB_PASSWORD`,
> `JBOARD_EVOLUTION_API_KEY` ou `JBOARD_API_TOKEN` estiverem vazios.

### 3. Subir a stack

```bash
cd /opt/jboard
docker compose -f infra/docker-compose.prod.yml --env-file infra/.env up -d --build
docker compose -f infra/docker-compose.prod.yml ps
```

Validar local (na VPS):

```bash
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8083/api/health   # 200
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8083/             # 200 (SPA)
```

### 4. Conectar o WhatsApp (Evolution API)

A Evolution sobe na `127.0.0.1:8084` (loopback, só setup). Criar a instância e
escanear o QR — da própria VPS ou via túnel SSH (`ssh -L 8084:127.0.0.1:8084 root@2.25.158.85`):

```bash
API_KEY=$(grep JBOARD_EVOLUTION_API_KEY /opt/jboard/infra/.env | cut -d= -f2)
curl -s -X POST http://127.0.0.1:8084/instance/create \
  -H "Content-Type: application/json" -H "apikey: $API_KEY" \
  -d '{"instanceName":"jboard","qrcode":true}'
# A resposta traz o base64 do QR. Reusar infra/setup-evolution.sh (aponta pra :8081
# em dev; em prod troque EVO_URL pra http://127.0.0.1:8084) ou os scripts fetch-qr.py.
```

Escanear: WhatsApp → Aparelhos conectados → Conectar aparelho. Status `open` =
conectado. Depois de conectado, o mapeamento `8084` pode ser removido do compose
(o backend fala com a Evolution pela rede interna `evolution-api:8080`).

### 5. Rota do Cloudflare Tunnel

Editar o config que o **systemd** usa (confirmar com `systemctl cat cloudflared | grep config`,
costuma ser `/etc/cloudflared/config.yml`). Adicionar **antes** do catch-all `404`:

```yaml
  - hostname: jboard.devarthur.com.br
    service: http://localhost:8083
  - service: http_status:404   # SEMPRE por último
```

Criar DNS, validar e reiniciar:

```bash
cloudflared tunnel route dns aad256c9-2a17-4be5-b1a2-67abbb007b50 jboard.devarthur.com.br
cloudflared tunnel --config /etc/cloudflared/config.yml ingress validate
systemctl restart cloudflared    # blip de ~3s em todos os sites do túnel
```

### 6. Validar de fora

```bash
curl -sI https://jboard.devarthur.com.br | head -1                       # 200
curl -s https://jboard.devarthur.com.br/api/health                       # {"status":"ok"}
# A API exige token; o SPA já o injeta no build. Direto:
curl -s -H "Authorization: Bearer <JBOARD_API_TOKEN>" \
  https://jboard.devarthur.com.br/api/boards                             # []
```

Abrir `https://jboard.devarthur.com.br` no navegador → kanban carrega e fala com
a API na mesma origem.

## Atualizar (redeploy)

```bash
cd /opt/jboard
git fetch --all && git reset --hard origin/main   # não "git pull" (ver playbook)
docker compose -f infra/docker-compose.prod.yml --env-file infra/.env up -d --build
```

> **Rotacionar o `JBOARD_API_TOKEN`** exige rebuild do `web` (o token é embutido
> no bundle do SPA em build-time) — o `--build` acima já cobre.

## Armadilhas

- **Porta 8083 livre?** Conferir o mapa de portas — colisão derruba o bind.
- **`/api` 502 atrás do Caddy:** backend não subiu ou DB não migrou. Ver
  `docker compose ... logs backend`.
- **Postgres exposto:** não há porta no host de propósito. Pra inspecionar,
  `docker compose ... exec postgres psql -U jboard`.
- **Token vazio = API aberta:** em prod o compose impede subir sem token, mas se
  rodar fora dele, garanta `JBOARD_API_TOKEN` setado.
```

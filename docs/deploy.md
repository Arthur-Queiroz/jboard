# jboard — Deploy na VPS (manual)

Deploy de produção na Hostinger KVM2 atrás do Cloudflare Tunnel, método
**Docker Compose + git pull** (ver `dev-notes/playbook-cicd-compose-cloudflare-tunnel`
e `dev-notes/deploy-kvm2-hostinger-cloudflared` no Obsidian). Este guia é o
deploy **manual**; o CI/CD via GitHub Actions vem por cima depois.

## Arquitetura

```
Internet → Cloudflare Edge → Tunnel (cloudflared) → 127.0.0.1:8084 → Caddy (web)
                                                                       ├─ /api/* → backend:8080 (Go)
                                                                       └─ /*     → SPA estático (Vue dist)
backend → postgres (dedicado)
backend → evolution-api (COMPARTILHADA, instância `inspire`, via rede n8n_inspiro_net)
```

**Subdomínio único** `jboard.devarthur.com.br`: o Caddy serve o SPA e faz
`reverse_proxy` de `/api` pro backend na **mesma origem** — sem CORS. Só o
serviço `web` abre porta, e no loopback (`127.0.0.1:8084`); Postgres e backend
só existem na rede interna do compose.

**WhatsApp via Evolution compartilhada.** O jboard NÃO sobe Evolution própria:
reusa a instância `inspire` da Evolution já rodando na VPS (projeto `n8n`, rede
`n8n_inspiro_net`), que já está conectada ao número do dono. O backend entra
nessa rede externa e fala com `evolution-api:8080`. A Evolution é só um gateway
(sem dado de negócio do jboard), então não há acoplamento de dados.

> Mapa de portas da VPS: 5678 n8n · 8080 Evolution(compartilhada) · 8081 jpad ·
> 8082/8443 jblog · 8083 jinitializr · **8084 jboard**.

## Pré-requisitos na VPS (uma vez)

A VPS já tem Docker + cloudflared (do jpad/jblog). Falta só a rota do tunnel
pro jboard. **A rota SSH e o sshd hardening já existem** (jpad/jblog) — não
mexer.

## Passo a passo

### 1. Clonar o repo na VPS

Repo precisa estar no GitHub (público, ou com deploy key se privado). Na VPS:

```bash
ssh root@187.127.34.186
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
# Evolution COMPARTILHADA: use a API key e a instância da Evolution que já roda
# na VPS (projeto n8n). Pegar a key: docker inspect evolution-api --format \
#   '{{range .Config.Env}}{{println .}}{{end}}' | grep AUTHENTICATION_API_KEY
JBOARD_EVOLUTION_API_KEY=<key-da-evolution-compartilhada>
JBOARD_EVOLUTION_INSTANCE=inspire                 # instância já conectada
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
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8084/api/health   # 200
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8084/             # 200 (SPA)
```

### 4. WhatsApp (Evolution compartilhada)

**Sem QR.** O jboard reusa a instância `inspire` da Evolution compartilhada, que
já está conectada. O backend só precisa estar na rede `n8n_inspiro_net` (o
compose.prod já declara essa rede como `external`) e ter no `.env` a API key +
instância da Evolution compartilhada (passo 2).

Confirme que a rede e a instância existem:

```bash
docker network ls | grep n8n_inspiro_net                      # a rede existe
KEY=$(docker inspect evolution-api --format '{{range .Config.Env}}{{println .}}{{end}}' | grep AUTHENTICATION_API_KEY | cut -d= -f2)
curl -s http://127.0.0.1:8080/instance/fetchInstances -H "apikey: $KEY" | grep -o '"name":"[^"]*"'   # deve listar "inspire"
```

Testar o envio de ponta a ponta: criar um lembrete com `reminder_at` ~70s no
futuro (via `POST /api/cards/{id}/reminders`) e conferir que o scheduler marca
`sent_at` e a mensagem chega no WhatsApp.

> Se um dia quiser uma instância dedicada (isolar do n8n), o `docker-compose.yml`
> de dev ainda traz Evolution + Redis próprios — basta portar esses serviços pro
> compose.prod e escanear um QR novo.

### 5. Rota do Cloudflare Tunnel

Editar o config que o **systemd** usa (confirmar com `systemctl cat cloudflared | grep config`,
costuma ser `/etc/cloudflared/config.yml`). Adicionar **antes** do catch-all `404`:

```yaml
  - hostname: jboard.devarthur.com.br
    service: http://localhost:8084
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

- **Porta 8084 livre?** Conferir o mapa de portas — colisão derruba o bind.
- **`/api` 502 atrás do Caddy:** backend não subiu ou DB não migrou. Ver
  `docker compose ... logs backend`.
- **Postgres exposto:** não há porta no host de propósito. Pra inspecionar,
  `docker compose ... exec postgres psql -U jboard`.
- **Token vazio = API aberta:** em prod o compose impede subir sem token, mas se
  rodar fora dele, garanta `JBOARD_API_TOKEN` setado.
```

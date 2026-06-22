# CLAUDE.md

Guia pro Claude Code (claude.ai/code) trabalhar neste repositório.

> **`AGENTS.md` na raiz é o guia canônico** (setup completo, project structure
> detalhado, code standards, gotchas). Este arquivo é o resumo operacional pro
> Claude; pra profundidade leia o `AGENTS.md`, `docs/architecture.md` e
> `docs/deploy.md`. **Mantenha os dois em sincronia** quando mudar algo estrutural.

## O que é

App pessoal de gestão: quadro **kanban** + **lembretes via WhatsApp** (substitui
Notion/Trello). Um só código-fonte **Vue 3** com dois builds (web e desktop
**Tauri**), backend **Go 1.26** (API REST chi + scheduler de lembretes), **Postgres**
dedicado e **Evolution API** (WhatsApp). Em produção, SPA e API são servidos na
mesma origem (`jboard.devarthur.com.br`) atrás de Caddy + Cloudflare Tunnel.

## Comandos

```bash
# Dependências locais (postgres + evolution-api)
docker compose -f infra/docker-compose.yml up -d postgres redis evolution-api

# Backend — http://localhost:8080
cd backend && JBOARD_DB_PASSWORD=jboard go run ./cmd/server

# Frontend — http://localhost:5173 (proxy /api -> :8080)
cd frontend && npm install && npm run dev

# Desktop (do frontend/) — dev abre janela com hot reload (API base = /api local)
npm run tauri:dev
# Release: o webview empacotado precisa da API base absoluta (sem proxy próprio)
VITE_JBOARD_API_BASE=https://jboard.devarthur.com.br/api \
  VITE_JBOARD_API_TOKEN=<token> npm run tauri:build
```

**Verificação antes de finalizar qualquer alteração** (rode sempre):

```bash
cd backend && gofmt -l . && go vet ./... && go build ./... && go test ./...
cd frontend && npm run build   # vue-tsc -b (typecheck) + vite build
```

- **Go fora do PATH do shell login:** use `/usr/local/go/bin/go` se `go` não resolver.
- **Um único teste Go:** `go test ./internal/api -run TestNome -v`.
- **`go test ./...` precisa de Docker** (testcontainers) e os de integração de DB
  (`db_integration_test.go`) exigem Postgres ativo. Sem Docker, rode os pacotes
  puros: `go test ./internal/api/...`.

## Estrutura (mapa rápido)

- `backend/cmd/server/main.go` — entrypoint: config → db → scheduler → HTTP.
- `backend/internal/api/` — handlers chi + `auth.go` (Bearer) + `validation.go` (400).
- `backend/internal/repository/` — interfaces + `Store` GORM (base pra mocks).
- `backend/internal/domain/` — models GORM: Board, Column, Card, Reminder.
- `backend/internal/db/` — conexão GORM + golang-migrate (`migrations/*.sql` via `//go:embed`).
- `backend/internal/scheduler/` — ticker 1min que dispara lembretes pendentes.
- `backend/internal/whatsapp/` — client da Evolution API (interface `Sender`).
- `frontend/src/` — Vue 3 + Vite + TS. `api.ts` é o client tipado; `App.vue` +
  `components/ColumnView.vue` montam o kanban com DnD (vue-draggable-plus).
- `desktop/src-tauri/` — shell Tauri (Rust): `tauri.conf.json`, `src/lib.rs`
  (tray icon, autostart, notificações).
- `infra/` — `docker-compose.yml` (dev), `docker-compose.prod.yml` (prod),
  `Dockerfile.web` + `Caddyfile` (SPA + `reverse_proxy /api`).
- `docs/architecture.md`, `docs/deploy.md` — arquitetura e runbook de deploy.

## Arquitetura (big picture)

- **Backend é a única fonte de verdade.** Web e desktop são clientes burros da
  mesma API REST. Camadas: `api/` → `repository/` → `domain/` → `db/`.

- **Scheduler dentro do processo Go.** Ticker de 1min varre lembretes pendentes e
  dispara via `whatsapp/`. Vive no backend (não no cliente) pra disparar mesmo com
  o desktop fechado.

- **Idempotência do lembrete via `sent_at`.** `Reminder.SentAt` (`*time.Time`) é
  nulo enquanto pendente. `MarkSent` faz `UPDATE ... WHERE sent_at IS NULL` — claim
  atômico contra duplicação entre ticks/instâncias. Envia **antes** de marcar: pior
  caso é mensagem duplicada, nunca lembrete perdido.

- **Migrations versionadas, não AutoMigrate.** golang-migrate com SQL embarcado.
  `Connect` roda `m.Up()` no boot (idempotente; `ErrNoChange` é OK). Schema change =
  nova migration `000N_nome.{up,down}.sql` — o `AutoMigrate` do GORM foi removido.

- **Auth por token Bearer.** Middleware em `api/auth.go`, **desligado** se
  `JBOARD_API_TOKEN` vazio. O frontend manda `Authorization: Bearer` quando
  `VITE_JBOARD_API_TOKEN` é definido no build.

- **Vue embutido no Tauri.** `tauri.conf.json` aponta `frontendDist` pra
  `../../frontend/dist` (UI no binário, não URL hospedada). Atualizar a UI do
  desktop exige rebuild do binário. O webview não tem proxy nem backend na própria
  origem: o build define `VITE_JBOARD_API_BASE` (URL absoluta) e o backend libera
  a origem `tauri://localhost` via `JBOARD_CORS_ORIGINS` (CORS em `api/cors.go`).

- **Sem Redis pro jboard.** Single-user, instância única: cache/fila/rate-limit não
  se justificam. O Redis no compose de dev existe só pra Evolution API.

## Convenções

- **Legibilidade acima de esperteza.** Go idiomático: `if err != nil` é padrão, não
  ruído. Early return ao invés de `if` aninhado; funções curtas; sem generics
  prematuros (escreva a versão concreta; use interface se só chama métodos).
- **Erros explícitos:** `repository.ErrNotFound` → 404; resto → 500
  (`respondRepoError`). `validation.go` retorna 400 com o campo inválido.
- **Comentários explicam o *porquê*** (decisões de negócio, pegadinhas de
  Postgres/GORM, workarounds), não o quê.
- **Vue:** `<script setup lang="ts">`, composables locais, sem estado global
  enquanto não for necessário. `api.ts` é o client tipado.

## Gotchas

- **Toolchain no WSL.** Go em `/usr/local/go/bin` (fora do PATH do shell login —
  use path absoluto). node/npm vêm do nvm (carrega no `~/.bashrc` interativo).
- **Evolution API.** O client assume `POST {baseURL}/message/sendText/{instance}`
  com header `apikey`. Validar o formato contra a versão da instância.
- **Deps de sistema do Tauri (Linux/WSL).** `cargo build` em `src-tauri` precisa de
  GTK/webkit **e `libayatana-appindicator3-dev`** (tray-icon). Instalar uma vez:
  ```
  sudo apt-get install -y libwebkit2gtk-4.1-dev build-essential curl wget file \
    libxdo-dev libssl-dev libayatana-appindicator3-dev librsvg2-dev pkg-config
  ```
  Sem elas o link falha em `glib-sys`/`gdk-sys`/`webkit2gtk-sys`/`appindicator-sys`.
- **Ícones do Tauri.** `npm run tauri:icon` (do `frontend/`) regera a partir de
  `desktop/src-tauri/icons/app-icon.svg`.

## Produção

Stack em `infra/docker-compose.prod.yml`: só `postgres` + `backend` + `web` (Caddy,
build do SPA + `reverse_proxy /api`). Ingress só via Cloudflare Tunnel → `web` em
`127.0.0.1:8084`; nenhuma porta aberta na VPS. **Reusa a Evolution compartilhada**
(instância `inspire`, rede externa `n8n_inspiro_net`), não sobe uma própria. Subir:
`docker compose -f infra/docker-compose.prod.yml --env-file infra/.env up -d --build`.
Runbook completo em `docs/deploy.md`.

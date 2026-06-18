# jboard

App pessoal de gestão (quadro kanban + lembretes via WhatsApp) pra substituir
Notion/Trello. Backend Go (API REST + scheduler de lembretes), frontend Vue 3
(único código-fonte, dois builds: web e desktop Tauri), Postgres dedicado e
Evolution API (WhatsApp). Deploy em VPS Hostinger KVM2 via Docker Compose
(git pull + build na VPS) + Caddy + Cloudflare Tunnel — subdomínio único
(`jboard.devarthur.com.br` serve SPA e API na mesma origem). Ver `docs/deploy.md`.

## Setup

```bash
# Banco + backend + Evolution API locais
docker compose -f infra/docker-compose.yml up -d

# Backend (Go) — http://localhost:8080
cd backend
JBOARD_DB_PASSWORD=jboard go run ./cmd/server

# Frontend (Vue) — http://localhost:5173 (proxy /api -> :8080)
cd frontend
npm install
npm run dev
```

## Build / Test / Lint

```bash
# Backend
cd backend
gofmt -l .              # deve listar nada
go vet ./...
go build ./...
go test ./...           # (suíte a escrever)

# Frontend
cd frontend
npm run build           # vue-tsc -b (typecheck) + vite build
npm run typecheck
```

Rode `gofmt`, `go vet` e `npm run build` antes de finalizar qualquer alteração.

## Project Structure

- `backend/cmd/server/main.go` — entrypoint: config, db, scheduler, HTTP server.
- `backend/internal/config/` — config por env (`JBOARD_*`), incluindo `JBOARD_API_TOKEN`.
- `backend/internal/db/` — conexão GORM + golang-migrate (migrations versionadas
  em `migrations/*.sql`, embarcadas no binário via `//go:embed`).
- `backend/internal/domain/` — models GORM: Board, Column, Card, Reminder.
- `backend/internal/repository/` — interfaces de acesso a dados + `Store` (GORM).
  Base pra mocks em teste. `MarkSent` é o claim atômico de `sent_at`.
- `backend/internal/api/` — handlers chi (CRUD de boards/columns/cards/reminders)
  + `auth.go` (middleware de token Bearer, desligado se `JBOARD_API_TOKEN` vazio).
  `validation.go` valida input (400 com mensagem do campo). Testes com fake
  in-memory em `api_test.go`/`fake_test.go`.
- `backend/internal/scheduler/` — ticker 1min: varre lembretes pendentes e dispara.
- `backend/internal/whatsapp/` — client da Evolution API (`Sender` interface).
- `frontend/src/` — Vue 3 + Vite + TS. `api.ts` é o client tipado (envia
  `Authorization: Bearer` se `VITE_JBOARD_API_TOKEN` definido em build); `App.vue`
  + `components/ColumnView.vue` montam o kanban com DnD (vue-draggable-plus).
  O frontend de produção é buildado e servido pelo `infra/Dockerfile.web` (Caddy).
- `desktop/src-tauri/` — shell Tauri (Rust). `tauri.conf.json` aponta
  `frontendDist` pra `../../frontend/dist` (UI embutida no binário).
  `src/lib.rs` tem tray icon (mostra/esconde janela, menu sair), autostart
  (inicia com o sistema) e plugin de notificação preparado. Ícones genéricos
  em `icons/` (quadrado azul com "j"), gerados via `npx tauri icon`.
- `infra/docker-compose.yml` — stack de DEV (postgres + redis + backend + evolution-api).
- `infra/docker-compose.prod.yml` — stack de PRODUÇÃO: sem portas no host (só
  `web` em `127.0.0.1:8084`), segredos via `infra/.env`, serviço `web` (Caddy)
  buildando o SPA + proxy `/api`.
- `infra/Dockerfile.web` — builda o Vue e serve dist/ + `reverse_proxy /api` (Caddy).
- `infra/Caddyfile` — front-door único: SPA estático + `/api/* → backend:8080`.
- `docs/architecture.md` — arquitetura (espelhar no Obsidian).
- `docs/deploy.md` — runbook de deploy manual na VPS.

## Architecture

**Backend é a única fonte de verdade.** API REST consumida por web e desktop;
o scheduler de lembretes roda dentro do processo Go (ticker 1min) porque precisa
disparar mesmo com o desktop fechado. Tudo no Postgres dedicado.

**Idempotência do lembrete via `sent_at`.** `Reminder.SentAt` é `*time.Time`
nulo enquanto pendente. O scheduler envia e depois chama `MarkSent`, que faz
`UPDATE ... WHERE sent_at IS NULL` — claim atômico que evita duplicação entre
ticks ou entre instâncias. Ordem enviar-antes-de-marcar: pior caso é uma
mensagem duplicada (se o MarkSent falhar), nunca um lembrete perdido.

**Vue embutido no Tauri.** Um só código-fonte Vue, dois builds. O desktop
embuta `frontend/dist` no binário (não aponta pra URL hospedada) — atualizações
de UI no desktop exigem rebuild do binário.

**Sem Redis.** Single-user, instância única: cache/fila/pubsub/rate-limit não
se justificam (ver `docs/architecture.md`). Voltam a fazer sentido só com
múltiplas instâncias do backend.

## Code Standards

- **Legibilidade acima de esperteza.** Go idiomático: `if err != nil` é padrão,
  não é ruído. Early return a `if` aninhado. Funções curtas com nomes claros.
- **Sem generics prematuros.** Escreva a versão concreta; só extraia generic
  com duplicação real de lógica idêntica entre tipos. Se só chama métodos, use
  interface.
- **Erros explícitos.** `repository.ErrNotFound` vira 404 no handler; resto
  vira 500 (`respondRepoError`).
- **Comentários explicam o porquê**, não o quê. Comente decisões de negócio,
  pegadinhas de Postgres/GORM e workarounds.
- **Vue:** `<script setup lang="ts">`, composables locais, sem estado global
  enquanto não for necessário.

## Gotchas

- **Toolchain dentro do WSL.** Toda a verificação (Go, npm, vite) roda nativa
  no WSL (ext4), sem cópia temporária nem atrito de UNC. Apenas uma ressalva:
  - **Go** está em `/usr/local/go/bin`, que não entra no PATH do shell login
    (`bash -l`). Use o path absoluto (`/usr/local/go/bin/go mod tidy`) ou
    adicione `export PATH="$PATH:/usr/local/go/bin"` ao `~/.profile`.
  - **node/npm** vêm do nvm, que carrega no `~/.bashrc` (shell interativo).
    `wsl bash -lc` (login não-interativo) não vê o nvm — use `wsl bash -ic`
    ou rode direto no terminal do WSL.
- **Migrations.** golang-migrate com migrations embarcadas (`internal/db/migrations/*.sql`).
  O `Connect` roda `m.Up()` no boot — idempotente (`ErrNoChange` é OK). Pra criar
  uma nova migration: adicione `0002_nome.up.sql` + `0002_nome.down.sql`. O
  `AutoMigrate` do GORM foi removido; qualquer schema change passa por migration.
- **Evolution API.** O client assume `POST {baseURL}/message/sendText/{instance}`
  com header `apikey`. Validar o formato exato contra a versão da instância.
- **Ícones do Tauri.** Gerados com `npx tauri icon` a partir de
  `desktop/src-tauri/icons/app-icon.png` (quadrado azul com "j"). Pra regerar
  com um logo real: substitua o PNG e rode `npm run tauri:icon` no `frontend/`.
- **Deps de sistema do Tauri (Linux/WSL).** O `cargo build` em `src-tauri`
  precisa das libs GTK/webkit. Instalar uma única vez:
  ```
  sudo apt-get install -y libwebkit2gtk-4.1-dev build-essential \
    curl wget file libxdo-dev libssl-dev libayatana-appindicator3-dev \
    librsvg2-dev pkg-config
  ```
  Sem essas libs, o `cargo check` falha em `glib-sys`/`gdk-sys`/`webkit2gtk-sys`
  (pkg-config não encontra as libs). As 472 crates Rust baixam normalmente;
  o erro é só na linkagem das bindings de sistema.
- **Build do desktop.** Após instalar as deps:
  ```
  cd frontend && npm run tauri:build    # release
  cd frontend && npm run tauri:dev      # dev (abre janela + hot reload Vue)
  ```
- **Deploy.** Ingress só via Cloudflare Tunnel → `web` (Caddy) em `127.0.0.1:8084`.
  Não abrir portas na VPS: no `docker-compose.prod.yml` só o `web` bind (loopback);
  postgres/redis/backend/evolution ficam na rede interna. Subir com
  `docker compose -f infra/docker-compose.prod.yml --env-file infra/.env up -d --build`.
  Runbook completo em `docs/deploy.md`.

## Manutenção deste arquivo

Mantenha o AGENTS.md atualizado quando mudanças no código exigirem.

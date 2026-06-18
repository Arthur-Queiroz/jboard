# jboard — Arquitetura

> Nome provisório, seguindo a convenção dos outros projetos pessoais (jpad, jblog). Renomeie à vontade antes de abrir o repositório de verdade.

## Contexto e motivação

App de gestão pessoal pra substituir Notion/Trello, que não funcionaram na prática. Duas features prioritárias:

1. Quadro kanban básico (boards → columns → cards).
2. Lembretes via WhatsApp para compromissos agendados (ex: avisar 5 minutos antes de uma aula), com WhatsApp escolhido como canal principal por ser visto com muito mais frequência que email.

## Stack e papéis de cada peça

| Componente | Tecnologia | Responsabilidade |
|---|---|---|
| Frontend web | Vue 3 + Vite | UI do kanban acessível via navegador |
| Frontend desktop | Tauri (Rust) + mesmo build do Vue | Cliente desktop nativo, UI embutida no binário |
| Backend | Go + GORM + chi | API REST, scheduler de lembretes, integração WhatsApp |
| Banco | Postgres (instância dedicada) | Persistência de boards/cards/reminders |
| WhatsApp | Evolution API (self-hosted) | Envio de mensagens via WhatsApp não-oficial |
| Deploy | Docker + Kamal 2 + Caddy + Cloudflare Tunnel | Hospedagem na Hostinger KVM2 |

A motivação de testar Rust foi o ponto de partida da escolha do Tauri — o objetivo explícito é aprender Rust na prática, sem precisar reescrever toda a lógica de negócio: o Rust fica restrito à camada nativa do desktop (tray icon, notificações, autostart), enquanto toda a lógica de produto vive no backend Go e na UI Vue.

## Por que backend e scheduler centralizados no Go

A decisão chave da arquitetura: onde mora o agendador dos lembretes. Como o lembrete precisa disparar independente do notebook estar aberto ou não, o scheduler não pode viver dentro do app Tauri — precisa ser um processo sempre ativo na VPS.

Por isso o Go backend é a única fonte de verdade: expõe a API REST consumida tanto pelo cliente web quanto pelo desktop, e roda internamente um scheduler (ticker de 1 minuto, ver `internal/scheduler`) que varre lembretes pendentes e dispara via Evolution API. Isso também resolve de cara o acesso multi-dispositivo, já que tanto web quanto desktop conversam com a mesma API.

## Banco de dados: por que Postgres (e não SQLite)

O desempate entre SQLite e Postgres foi feito contra cinco prioridades, em ordem de importância:

1. **Facilidade na hora de codificar** — leve vantagem pro Postgres: tipos nativos mais ricos (JSONB, arrays) e, principalmente, evita o problema de locking do SQLite (`database is locked`) quando o scheduler (ticker) e os handlers HTTP acessam o banco concorrentemente, cenário que essa arquitetura tem desde o dia 1.
2. **Facilidade pro deploy** — isoladamente o SQLite seria mais simples (sem serviço de rede), mas o driver Postgres do GORM (`gorm.io/driver/postgres`, baseado em `pgx`) é pure-Go, sem CGO, então não há a fricção de build que existiria com o driver padrão do SQLite (`mattn/go-sqlite3`, baseado em CGO).
3. **Facilidade pra manter** — Postgres ganha por permitir inspeção remota (psql, DBeaver) sem precisar puxar arquivo da VPS, e por lidar melhor com múltiplos processos conforme o projeto crescer (CLI futura, novas integrações).
4. **Facilidade pra escrever testes** — único critério onde SQLite venceria de forma clara (banco em memória, sem dependência de container), mas mitigado com `testcontainers-go` pra testes de integração e interfaces de repositório com mocks pra testes unitários.
5. **Performance** — irrelevante pra esse projeto (single-user), não decide nada.

Como os critérios 1, 2 e 3 já pendem pro Postgres, e são os de maior peso na ordem definida, Postgres venceu o desempate.

### Por que instância dedicada (e não compartilhada com o jblog)

Embora reaproveitar a instância de Postgres do jblog (que já roda via Kamal 2 na KVM2 atual) reduzisse a infraestrutura nova necessária, a decisão final foi por uma instância dedicada — motivada pela migração planejada da VPS para uma KVM2 nova (Brasil, custo menor) após o dia 25. Isolar o banco agora evita ter que desentrelaçar dump/restore de dois projetos na hora da migração; o jboard pode ser movido independentemente do jblog.

## Por que não Redis

Avaliados os usos clássicos de Redis (cache, fila de jobs, pub/sub, idempotência, rate limiting) contra a escala real do projeto (single-user, instância única do backend), nenhum se justifica:

- Cache de leitura: Postgres já responde em microssegundos pra um usuário só.
- Fila de jobs: um ticker simples no Go cobre o volume de lembretes pessoais sem precisar de fila distribuída.
- Pub/sub pra real-time entre web/desktop: resolvido dentro do próprio processo Go (channels/goroutines), ou via `LISTEN/NOTIFY` nativo do Postgres se necessário — não há múltiplas instâncias do backend pra coordenar.
- Idempotência de envio: resolvido com uma coluna `sent_at` no Postgres, sem necessidade de lock distribuído (e há `pg_advisory_lock` no Postgres se algum dia precisar).
- Rate limiting: em memória no próprio processo, já que só existe uma instância do backend.

Redis voltaria a fazer sentido se o projeto crescesse pra múltiplas instâncias do backend em paralelo ou um volume de jobs muito maior que lembretes pessoais — não é o cenário atual.

## WhatsApp via Evolution API

Mesma abordagem usada no projeto da Inspire Pilates: Evolution API (Baileys) na VPS. Como é uso pessoal, single-recipient, com intervalos de horas entre mensagens, o perfil de risco de ban é baixo — os padrões de detecção giram em torno de comportamento de spam/bulk, não desse uso. Vale considerar um número secundário pra essa automação se quiser eliminar qualquer risco residual pro número principal.

**Atualização (deploy real, 2026-06-18):** em produção o jboard NÃO sobe Evolution dedicada — reusa a instância `inspire` da Evolution compartilhada que já roda na VPS (projeto n8n), pela rede `n8n_inspiro_net`. A decisão original era por instância dedicada, mas a Evolution é só um gateway de WhatsApp (não guarda dado de negócio do jboard, que vive todo no Postgres dedicado), então compartilhar a instância já conectada evita um 2º aparelho/QR e containers redundantes, sem acoplar nada que dificulte a futura migração de VPS (basta repontar a URL). O Postgres segue dedicado. Em dev, o `docker-compose.yml` ainda sobe Evolution + Redis próprios.

## Tauri: Vue embutido no binário

Decisão: o Vue é embutido no binário (`frontendDist` aponta pra `frontend/dist` no `tauri.conf.json`), não uma webview apontando pra URL hospedada. Isso significa um único código-fonte Vue compartilhado entre web e desktop, mas dois processos de build distintos — atualizações de UI no desktop exigem rebuild do binário (considerar auto-update do Tauri no futuro, se a fricção de reinstalar manualmente incomodar).

## CLI futura

Quando a CLI for implementada, ela deve ser um client HTTP da mesma API REST do backend — não deve abrir o arquivo do banco diretamente. Isso evita contenção de múltiplos processos escrevendo no Postgres ao mesmo tempo e mantém o backend como única fonte de verdade, mesmo sendo Postgres (que tecnicamente suporta múltiplos writers) — é uma escolha de consistência arquitetural, não uma limitação técnica do banco.

## Estrutura do repositório

```
jboard/
├── backend/              # Go: API REST + scheduler + client Evolution API
│   ├── cmd/server/
│   └── internal/
│       ├── config/
│       ├── db/
│       ├── domain/       # models GORM (Board, Column, Card, Reminder)
│       ├── repository/   # camada de acesso a dados (interfaces p/ mocks em teste)
│       ├── api/          # handlers HTTP (chi)
│       ├── scheduler/    # ticker de verificação de lembretes
│       └── whatsapp/     # client Evolution API
├── frontend/             # Vue 3 + Vite — único código-fonte, dois builds
├── desktop/
│   └── src-tauri/        # shell nativo (Rust), aponta frontendDist pra ../../frontend/dist
├── infra/
│   ├── docker-compose.yml  # postgres dedicado + backend + evolution-api
│   └── Caddyfile
└── docs/                 # esse diretório — espelhar no Obsidian
```

## Próximos passos sugeridos

- Implementar os handlers de `internal/api` (CRUD de boards/columns/cards/reminders).
- Implementar a query real do scheduler (lembretes pendentes + marcação atômica de `sent_at`).
- Decidir biblioteca de migrations complementar ao `AutoMigrate` do GORM (golang-migrate ou goose) antes de qualquer alteração destrutiva de schema em produção.
- Configurar a instância da Evolution API e testar o fluxo de QR code / sessão.
- Configurar Cloudflare Tunnel + Caddy pros dois subdomínios (`jboard-api` e `jboard`) na KVM2 nova, pós-migração.

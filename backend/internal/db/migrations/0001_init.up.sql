CREATE TABLE IF NOT EXISTS boards (
    id           BIGSERIAL PRIMARY KEY,
    title        TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS columns (
    id           BIGSERIAL PRIMARY KEY,
    board_id     BIGINT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    position     INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cards (
    id           BIGSERIAL PRIMARY KEY,
    column_id    BIGINT NOT NULL REFERENCES columns(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    position     INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reminders (
    id           BIGSERIAL PRIMARY KEY,
    card_id      BIGINT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    reminder_at  TIMESTAMPTZ NOT NULL,
    recipient    TEXT NOT NULL DEFAULT '',
    message      TEXT NOT NULL DEFAULT '',
    sent_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_columns_board_id ON columns(board_id);
CREATE INDEX IF NOT EXISTS idx_cards_column_id ON cards(column_id);
CREATE INDEX IF NOT EXISTS idx_reminders_card_id ON reminders(card_id);
CREATE INDEX IF NOT EXISTS idx_reminders_pending ON reminders(reminder_at) WHERE sent_at IS NULL;

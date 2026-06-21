CREATE TYPE import_batch_status AS ENUM ('open', 'closed');
CREATE TYPE moderation_row_status AS ENUM ('pending', 'ready', 'error');

CREATE TABLE import_batches (
    id          BIGSERIAL PRIMARY KEY,
    file_name   VARCHAR(500) NOT NULL,
    total_rows  INTEGER NOT NULL DEFAULT 0,
    status      import_batch_status NOT NULL DEFAULT 'open',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at   TIMESTAMPTZ
);

CREATE TABLE moderation_transactions (
    id           BIGSERIAL PRIMARY KEY,
    batch_id     BIGINT NOT NULL REFERENCES import_batches (id) ON DELETE CASCADE,
    row_number   INTEGER NOT NULL,
    title        VARCHAR(500),
    amount       BIGINT,
    date         DATE,
    tag_id       BIGINT REFERENCES tags (id),
    category     transaction_category,
    specificity  transaction_specificity,
    comment      TEXT,
    url          TEXT,
    status       moderation_row_status NOT NULL DEFAULT 'pending',
    field_errors JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_moderation_transactions_batch_id ON moderation_transactions (batch_id);

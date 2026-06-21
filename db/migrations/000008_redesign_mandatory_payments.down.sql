-- enum values cannot be removed in PostgreSQL — skip reverting payment_recurrence

DROP TABLE IF EXISTS mandatory_payments;

CREATE TABLE mandatory_payments (
    id           BIGSERIAL PRIMARY KEY,
    title        VARCHAR(500)       NOT NULL,
    amount       BIGINT             NOT NULL CHECK (amount > 0),
    tag_id       BIGINT             NOT NULL REFERENCES tags (id),
    recurrence   payment_recurrence NOT NULL,
    due_day      SMALLINT           NOT NULL CHECK (due_day BETWEEN 1 AND 31),
    created_at   TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ        NOT NULL DEFAULT NOW()
);

CREATE TABLE mandatory_payment_statuses (
    id                    BIGSERIAL PRIMARY KEY,
    mandatory_payment_id  BIGINT  NOT NULL REFERENCES mandatory_payments (id) ON DELETE CASCADE,
    year                  SMALLINT NOT NULL,
    month                 SMALLINT NOT NULL CHECK (month BETWEEN 1 AND 12),
    paid                  BOOLEAN  NOT NULL DEFAULT FALSE,
    UNIQUE (mandatory_payment_id, year, month)
);

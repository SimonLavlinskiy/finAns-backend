ALTER TYPE payment_recurrence ADD VALUE IF NOT EXISTS 'daily';
ALTER TYPE payment_recurrence ADD VALUE IF NOT EXISTS 'weekly';
ALTER TYPE payment_recurrence ADD VALUE IF NOT EXISTS 'semi_annual';

DROP TABLE IF EXISTS mandatory_payment_statuses;
DROP TABLE IF EXISTS mandatory_payments;

CREATE TABLE mandatory_payments (
    id                 BIGSERIAL PRIMARY KEY,
    title              VARCHAR(500)       NOT NULL,
    amount             BIGINT             NOT NULL CHECK (amount > 0),
    tag_id             BIGINT             NOT NULL REFERENCES tags (id),
    recurrence         payment_recurrence NOT NULL,
    next_payment_date  DATE               NOT NULL,
    created_at         TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ        NOT NULL DEFAULT NOW()
);

CREATE INDEX ON mandatory_payments (next_payment_date);

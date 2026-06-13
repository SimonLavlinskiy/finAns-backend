-- finAns initial schema (TZ v1.0)

CREATE TYPE transaction_category AS ENUM ('expense', 'income');
CREATE TYPE transaction_specificity AS ENUM ('required', 'simple');
CREATE TYPE payment_recurrence AS ENUM ('monthly', 'quarterly', 'yearly');
CREATE TYPE payment_status AS ENUM ('pending', 'paid', 'overdue');
CREATE TYPE limit_period_type AS ENUM ('week', 'month', 'custom');

CREATE TABLE tags (
    id          BIGSERIAL PRIMARY KEY,
    parent_id   BIGINT REFERENCES tags (id) ON DELETE SET NULL,
    name        VARCHAR(255) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tags_parent_id ON tags (parent_id);

CREATE TABLE transactions (
    id              BIGSERIAL PRIMARY KEY,
    title           VARCHAR(500) NOT NULL,
    amount          NUMERIC(15, 2) NOT NULL CHECK (amount >= 0),
    date            DATE NOT NULL,
    tag_id          BIGINT NOT NULL REFERENCES tags (id),
    category        transaction_category NOT NULL,
    specificity     transaction_specificity NOT NULL,
    comment         TEXT,
    url             TEXT,
    file_path       TEXT,
    file_name       TEXT,
    file_mime_type  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_date ON transactions (date);
CREATE INDEX idx_transactions_tag_id ON transactions (tag_id);
CREATE INDEX idx_transactions_category ON transactions (category);

CREATE TABLE spending_limits (
    id              BIGSERIAL PRIMARY KEY,
    tag_id          BIGINT NOT NULL REFERENCES tags (id),
    amount          NUMERIC(15, 2) NOT NULL CHECK (amount > 0),
    period_type     limit_period_type NOT NULL,
    period_start    DATE,
    period_end      DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE mandatory_payments (
    id          BIGSERIAL PRIMARY KEY,
    title       VARCHAR(500) NOT NULL,
    amount      NUMERIC(15, 2) NOT NULL CHECK (amount >= 0),
    tag_id      BIGINT NOT NULL REFERENCES tags (id),
    recurrence  payment_recurrence NOT NULL,
    due_day     SMALLINT NOT NULL CHECK (due_day BETWEEN 1 AND 31),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE mandatory_payment_statuses (
    id                    BIGSERIAL PRIMARY KEY,
    mandatory_payment_id  BIGINT NOT NULL REFERENCES mandatory_payments (id) ON DELETE CASCADE,
    year                  SMALLINT NOT NULL,
    month                 SMALLINT NOT NULL CHECK (month BETWEEN 1 AND 12),
    status                payment_status NOT NULL DEFAULT 'pending',
    paid_at               TIMESTAMPTZ,
    transaction_id        BIGINT REFERENCES transactions (id) ON DELETE SET NULL,
    UNIQUE (mandatory_payment_id, year, month)
);

CREATE TABLE planned_expenses (
    id            BIGSERIAL PRIMARY KEY,
    title         VARCHAR(500) NOT NULL,
    amount        NUMERIC(15, 2) NOT NULL CHECK (amount >= 0),
    planned_date  DATE NOT NULL,
    tag_id        BIGINT NOT NULL REFERENCES tags (id),
    comment       TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_balance (
    id              BIGSERIAL PRIMARY KEY,
    initial_amount  NUMERIC(15, 2) NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO user_balance (initial_amount) VALUES (0);

DROP TABLE planned_expenses;
DROP TABLE planned_expense_categories;
DROP TYPE planned_expense_status;
DROP TYPE planned_expense_priority;

CREATE TABLE planned_expenses (
    id            BIGSERIAL PRIMARY KEY,
    title         VARCHAR(500) NOT NULL,
    amount        NUMERIC(15, 2) NOT NULL CHECK (amount >= 0),
    planned_date  DATE NOT NULL,
    tag_id        BIGINT NOT NULL REFERENCES tags (id),
    comment       TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DROP TABLE planned_expenses;

CREATE TYPE planned_expense_priority AS ENUM ('low', 'medium', 'high');
CREATE TYPE planned_expense_status AS ENUM ('active', 'archived');

CREATE TABLE planned_expense_categories (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(200) NOT NULL,
    color       VARCHAR(7) NOT NULL,
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE planned_expenses (
    id             BIGSERIAL PRIMARY KEY,
    category_id    BIGINT NOT NULL REFERENCES planned_expense_categories (id) ON DELETE CASCADE,
    title          VARCHAR(500) NOT NULL,
    cost_kopecks   BIGINT CHECK (cost_kopecks >= 0),
    due_date       DATE,
    url            TEXT,
    priority       planned_expense_priority NOT NULL DEFAULT 'medium',
    status         planned_expense_status NOT NULL DEFAULT 'active',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at    TIMESTAMPTZ
);

BEGIN;

-- 1. Теги: мягкое удаление, вид категории и порядок
ALTER TABLE tags ADD COLUMN deleted_at TIMESTAMPTZ NULL;
ALTER TABLE tags ADD COLUMN kind VARCHAR(10) NOT NULL DEFAULT 'expense';
ALTER TABLE tags ADD CONSTRAINT tags_kind_check CHECK (kind IN ('expense', 'income'));
ALTER TABLE tags ADD COLUMN sort_order INT NOT NULL DEFAULT 0;

-- kind=income: корневой тег, у поддерева которого есть операции и все они доходные
WITH RECURSIVE subtree AS (
    SELECT id AS root_id, id FROM tags WHERE parent_id IS NULL
    UNION ALL
    SELECT s.root_id, t.id FROM tags t JOIN subtree s ON t.parent_id = s.id
),
income_roots AS (
    SELECT s.root_id
    FROM subtree s JOIN transactions tr ON tr.tag_id = s.id
    GROUP BY s.root_id
    HAVING BOOL_AND(tr.category = 'income'::transaction_category)
)
UPDATE tags SET kind = 'income' WHERE id IN (SELECT root_id FROM income_roots);

-- подтеги наследуют kind корня
WITH RECURSIVE subtree AS (
    SELECT id AS root_id, id FROM tags WHERE parent_id IS NULL
    UNION ALL
    SELECT s.root_id, t.id FROM tags t JOIN subtree s ON t.parent_id = s.id
)
UPDATE tags t SET kind = r.kind
FROM subtree s JOIN tags r ON r.id = s.root_id
WHERE t.id = s.id AND t.parent_id IS NOT NULL;

-- порядок: как сейчас отображается (по алфавиту внутри родителя)
UPDATE tags t SET sort_order = o.rn
FROM (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY project_id, parent_id ORDER BY name) AS rn
    FROM tags
) o
WHERE t.id = o.id;

CREATE INDEX idx_tags_project_alive ON tags (project_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_transactions_project_date ON transactions (project_id, date);

-- 2. Лимиты: один месячный лимит на категорию
CREATE TABLE spending_limits_dedup_backup AS
SELECT sl.* FROM spending_limits sl
WHERE sl.period_type = 'month'
  AND EXISTS (
      SELECT 1 FROM spending_limits o
      WHERE o.period_type = 'month'
        AND o.project_id IS NOT DISTINCT FROM sl.project_id
        AND o.tag_id = sl.tag_id
        AND (o.created_at, o.id) > (sl.created_at, sl.id)
  );

DELETE FROM spending_limits WHERE id IN (SELECT id FROM spending_limits_dedup_backup);

CREATE UNIQUE INDEX ux_spending_limits_month
    ON spending_limits (project_id, tag_id) WHERE period_type = 'month';

-- 3. Планирование: одно на проект
CREATE TABLE planners (
    project_id  BIGINT PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    data        JSONB NOT NULL,
    updated_by  BIGINT REFERENCES users (id),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;

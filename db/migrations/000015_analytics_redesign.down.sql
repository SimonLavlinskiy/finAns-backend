BEGIN;

-- ВНИМАНИЕ: после отката мягко удалённые теги (deleted_at IS NOT NULL) снова
-- становятся видимыми как обычные — информация об удалении теряется.

DROP TABLE IF EXISTS planners;

DROP INDEX IF EXISTS ux_spending_limits_month;
INSERT INTO spending_limits SELECT * FROM spending_limits_dedup_backup;
DROP TABLE IF EXISTS spending_limits_dedup_backup;

DROP INDEX IF EXISTS idx_transactions_project_date;
DROP INDEX IF EXISTS idx_tags_project_alive;

ALTER TABLE tags DROP COLUMN IF EXISTS sort_order;
ALTER TABLE tags DROP CONSTRAINT IF EXISTS tags_kind_check;
ALTER TABLE tags DROP COLUMN IF EXISTS kind;
ALTER TABLE tags DROP COLUMN IF EXISTS deleted_at;

COMMIT;

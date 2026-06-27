BEGIN;

-- Remove project_id from all data tables
ALTER TABLE planned_expenses          DROP COLUMN IF EXISTS project_id;
ALTER TABLE planned_expense_categories DROP COLUMN IF EXISTS project_id;
ALTER TABLE user_balance              DROP COLUMN IF EXISTS project_id;
ALTER TABLE mandatory_payments        DROP COLUMN IF EXISTS project_id;
ALTER TABLE spending_limits           DROP COLUMN IF EXISTS project_id;
ALTER TABLE transactions              DROP COLUMN IF EXISTS project_id;
ALTER TABLE tags                      DROP COLUMN IF EXISTS project_id;

-- Drop project tables
DROP TABLE IF EXISTS project_members;
DROP TABLE IF EXISTS projects;

-- Restore users table to login/password schema (passwords cannot be recovered)
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_key;
ALTER TABLE users ADD COLUMN login         VARCHAR(64) UNIQUE;
ALTER TABLE users ADD COLUMN password_hash TEXT NOT NULL DEFAULT '';

UPDATE users SET login = username;

ALTER TABLE users ALTER COLUMN login SET NOT NULL;
ALTER TABLE users DROP COLUMN username;
ALTER TABLE users DROP COLUMN display_name;

COMMIT;

BEGIN;

-- Step 1: Evolve users table (replace login/password auth with username/display_name)
ALTER TABLE users ADD COLUMN username     VARCHAR(20);
ALTER TABLE users ADD COLUMN display_name VARCHAR(100);

UPDATE users SET username = login, display_name = initcap(login);

ALTER TABLE users ALTER COLUMN username     SET NOT NULL;
ALTER TABLE users ALTER COLUMN display_name SET NOT NULL;
ALTER TABLE users ADD CONSTRAINT users_username_key UNIQUE (username);

ALTER TABLE users DROP COLUMN login;
ALTER TABLE users DROP COLUMN password_hash;

-- Step 2: Create projects table
CREATE TABLE projects (
    id                      BIGSERIAL PRIMARY KEY,
    name                    VARCHAR(200) NOT NULL,
    initial_balance_kopecks BIGINT NOT NULL DEFAULT 0,
    started_at              DATE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Step 3: Create project_members table
CREATE TABLE project_members (
    project_id  BIGINT NOT NULL REFERENCES projects(id),
    user_id     BIGINT NOT NULL REFERENCES users(id),
    role        VARCHAR(10) NOT NULL DEFAULT 'member',
    PRIMARY KEY (project_id, user_id)
);

-- Step 4: Insert default project (copy initial_balance from user_balance)
INSERT INTO projects (name, initial_balance_kopecks)
SELECT 'Мои финансы', CAST(COALESCE((SELECT initial_amount FROM user_balance ORDER BY id LIMIT 1), 0) AS BIGINT);

-- Step 5: Add all existing users as project members (simon = owner, others = member)
INSERT INTO project_members (project_id, user_id, role)
SELECT
    (SELECT id FROM projects WHERE name = 'Мои финансы'),
    u.id,
    CASE WHEN u.username = 'simon' THEN 'owner' ELSE 'member' END
FROM users u;

-- Step 6: Add project_id to all data tables (nullable first for backfill)
ALTER TABLE tags                      ADD COLUMN project_id BIGINT REFERENCES projects(id);
ALTER TABLE transactions              ADD COLUMN project_id BIGINT REFERENCES projects(id);
ALTER TABLE spending_limits           ADD COLUMN project_id BIGINT REFERENCES projects(id);
ALTER TABLE mandatory_payments        ADD COLUMN project_id BIGINT REFERENCES projects(id);
ALTER TABLE user_balance              ADD COLUMN project_id BIGINT REFERENCES projects(id);
ALTER TABLE planned_expense_categories ADD COLUMN project_id BIGINT REFERENCES projects(id);
ALTER TABLE planned_expenses          ADD COLUMN project_id BIGINT REFERENCES projects(id);

-- Step 7: Backfill project_id for all existing data
UPDATE tags                      SET project_id = (SELECT id FROM projects WHERE name = 'Мои финансы');
UPDATE transactions              SET project_id = (SELECT id FROM projects WHERE name = 'Мои финансы');
UPDATE spending_limits           SET project_id = (SELECT id FROM projects WHERE name = 'Мои финансы');
UPDATE mandatory_payments        SET project_id = (SELECT id FROM projects WHERE name = 'Мои финансы');
UPDATE user_balance              SET project_id = (SELECT id FROM projects WHERE name = 'Мои финансы');
UPDATE planned_expense_categories SET project_id = (SELECT id FROM projects WHERE name = 'Мои финансы');
UPDATE planned_expenses          SET project_id = (SELECT id FROM projects WHERE name = 'Мои финансы');

-- Step 8: Enforce NOT NULL and unique balance per project
ALTER TABLE tags                      ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE transactions              ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE spending_limits           ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE mandatory_payments        ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE user_balance              ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE user_balance              ADD CONSTRAINT user_balance_project_id_key UNIQUE (project_id);
ALTER TABLE planned_expense_categories ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE planned_expenses          ALTER COLUMN project_id SET NOT NULL;

COMMIT;

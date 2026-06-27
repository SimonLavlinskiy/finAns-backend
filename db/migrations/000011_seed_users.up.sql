BEGIN;

-- Create users simon and anna if they don't already exist
INSERT INTO users (username, display_name)
VALUES ('simon', 'Simon'), ('anna', 'Anna')
ON CONFLICT (username) DO NOTHING;

-- Ensure the default project exists
INSERT INTO projects (name, initial_balance_kopecks)
SELECT 'Мои финансы', COALESCE((SELECT initial_amount FROM user_balance ORDER BY id LIMIT 1), 0)
WHERE NOT EXISTS (SELECT 1 FROM projects WHERE name = 'Мои финансы');

-- Add simon as owner (if not already a member)
INSERT INTO project_members (project_id, user_id, role)
SELECT p.id, u.id, 'owner'
FROM projects p, users u
WHERE p.name = 'Мои финансы' AND u.username = 'simon'
ON CONFLICT (project_id, user_id) DO UPDATE SET role = 'owner';

-- Add anna as member (if not already a member)
INSERT INTO project_members (project_id, user_id, role)
SELECT p.id, u.id, 'member'
FROM projects p, users u
WHERE p.name = 'Мои финансы' AND u.username = 'anna'
ON CONFLICT (project_id, user_id) DO NOTHING;

-- Backfill project_id for any data rows still missing it
UPDATE tags SET project_id = (SELECT id FROM projects WHERE name = 'Мои финансы')
WHERE project_id IS NULL;
UPDATE transactions SET project_id = (SELECT id FROM projects WHERE name = 'Мои финансы')
WHERE project_id IS NULL;
UPDATE spending_limits SET project_id = (SELECT id FROM projects WHERE name = 'Мои финансы')
WHERE project_id IS NULL;
UPDATE mandatory_payments SET project_id = (SELECT id FROM projects WHERE name = 'Мои финансы')
WHERE project_id IS NULL;
UPDATE user_balance SET project_id = (SELECT id FROM projects WHERE name = 'Мои финансы')
WHERE project_id IS NULL;
UPDATE planned_expense_categories SET project_id = (SELECT id FROM projects WHERE name = 'Мои финансы')
WHERE project_id IS NULL;
UPDATE planned_expenses SET project_id = (SELECT id FROM projects WHERE name = 'Мои финансы')
WHERE project_id IS NULL;

COMMIT;

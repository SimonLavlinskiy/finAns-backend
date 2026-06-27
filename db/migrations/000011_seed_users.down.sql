BEGIN;

DELETE FROM project_members
WHERE user_id IN (SELECT id FROM users WHERE username IN ('simon', 'anna'));

DELETE FROM users WHERE username IN ('simon', 'anna');

COMMIT;

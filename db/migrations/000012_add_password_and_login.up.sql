BEGIN;

-- Restore password_hash column
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);

-- Set passwords for existing users
UPDATE users SET password_hash = '$2a$10$qULeviBXokVlBGLJmgujzOOvOMvcmBzCPwpu048NdzcVGxnLlsYaq' WHERE username = 'simon';
UPDATE users SET password_hash = '$2a$10$QEDJxjJkOIP9vpsDhXx2WeLrrV2DkCkPZKNWGJzL85pi1IpbZ8uLS' WHERE username = 'anna';

-- Make it NOT NULL
ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;

-- Mark simon as admin
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE users SET is_admin = TRUE WHERE username = 'simon';

COMMIT;

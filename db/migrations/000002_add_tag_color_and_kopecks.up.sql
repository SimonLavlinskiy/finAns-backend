ALTER TABLE tags ADD COLUMN IF NOT EXISTS color VARCHAR(7) NOT NULL DEFAULT '#94A3B8';

ALTER TABLE transactions ADD COLUMN amount_kopecks BIGINT;
UPDATE transactions SET amount_kopecks = (amount * 100)::BIGINT WHERE amount_kopecks IS NULL;
ALTER TABLE transactions DROP COLUMN amount;
ALTER TABLE transactions RENAME COLUMN amount_kopecks TO amount;
ALTER TABLE transactions ALTER COLUMN amount SET NOT NULL;
ALTER TABLE transactions ADD CONSTRAINT transactions_amount_check CHECK (amount >= 0);

ALTER TABLE user_balance ADD COLUMN initial_amount_kopecks BIGINT;
UPDATE user_balance SET initial_amount_kopecks = (initial_amount * 100)::BIGINT WHERE initial_amount_kopecks IS NULL;
ALTER TABLE user_balance DROP COLUMN initial_amount;
ALTER TABLE user_balance RENAME COLUMN initial_amount_kopecks TO initial_amount;
ALTER TABLE user_balance ALTER COLUMN initial_amount SET DEFAULT 0;
ALTER TABLE user_balance ALTER COLUMN initial_amount SET NOT NULL;

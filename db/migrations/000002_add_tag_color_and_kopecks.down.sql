ALTER TABLE tags DROP COLUMN IF EXISTS color;

ALTER TABLE user_balance ADD COLUMN initial_amount_decimal NUMERIC(15, 2);
UPDATE user_balance SET initial_amount_decimal = initial_amount / 100.0;
ALTER TABLE user_balance DROP COLUMN initial_amount;
ALTER TABLE user_balance RENAME COLUMN initial_amount_decimal TO initial_amount;
ALTER TABLE user_balance ALTER COLUMN initial_amount SET DEFAULT 0;

ALTER TABLE transactions ADD COLUMN amount_decimal NUMERIC(15, 2);
UPDATE transactions SET amount_decimal = amount / 100.0;
ALTER TABLE transactions DROP COLUMN amount;
ALTER TABLE transactions RENAME COLUMN amount_decimal TO amount;
ALTER TABLE transactions ADD CONSTRAINT transactions_amount_check CHECK (amount >= 0);

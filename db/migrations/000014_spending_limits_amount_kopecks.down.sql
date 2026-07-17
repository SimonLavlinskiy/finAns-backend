ALTER TABLE spending_limits DROP CONSTRAINT spending_limits_amount_check;
ALTER TABLE spending_limits ADD COLUMN amount_rubles NUMERIC(15, 2);
UPDATE spending_limits SET amount_rubles = (amount / 100.0);
ALTER TABLE spending_limits ALTER COLUMN amount_rubles SET NOT NULL;
ALTER TABLE spending_limits DROP COLUMN amount;
ALTER TABLE spending_limits RENAME COLUMN amount_rubles TO amount;
ALTER TABLE spending_limits ADD CONSTRAINT spending_limits_amount_check CHECK (amount > 0);

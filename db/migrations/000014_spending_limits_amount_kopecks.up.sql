ALTER TABLE spending_limits ADD COLUMN amount_kopecks BIGINT;
UPDATE spending_limits SET amount_kopecks = (amount * 100)::BIGINT;
ALTER TABLE spending_limits ALTER COLUMN amount_kopecks SET NOT NULL;
ALTER TABLE spending_limits DROP COLUMN amount;
ALTER TABLE spending_limits RENAME COLUMN amount_kopecks TO amount;
ALTER TABLE spending_limits ADD CONSTRAINT spending_limits_amount_check CHECK (amount > 0);

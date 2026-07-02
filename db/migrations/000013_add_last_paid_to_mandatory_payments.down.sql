BEGIN;

ALTER TABLE mandatory_payments
    DROP COLUMN IF EXISTS last_paid_at,
    DROP COLUMN IF EXISTS last_transaction_id;

COMMIT;

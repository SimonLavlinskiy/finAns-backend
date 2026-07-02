BEGIN;

ALTER TABLE mandatory_payments
    ADD COLUMN last_paid_at        TIMESTAMPTZ,
    ADD COLUMN last_transaction_id BIGINT;

COMMIT;

BEGIN;

DROP INDEX transactions_idempotency_keys;
DROP INDEX messages_idempotency_keys;

ALTER TABLE transactions DROP COLUMN idempotency_key;
ALTER TABLE messages DROP COLUMN idempotency_key;

COMMIT;

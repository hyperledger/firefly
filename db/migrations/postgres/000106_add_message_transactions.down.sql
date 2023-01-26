BEGIN;
ALTER TABLE messages DROP COLUMN tx_batch;
ALTER TABLE messages DROP COLUMN tx_related;
COMMIT;

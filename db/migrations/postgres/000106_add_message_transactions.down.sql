BEGIN;
ALTER TABLE messages DROP COLUMN tx_id;
ALTER TABLE messages DROP COLUMN tx_parent_type;
ALTER TABLE messages DROP COLUMN tx_parent_id;
COMMIT;

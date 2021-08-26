BEGIN;
ALTER TABLE tokenpool DROP COLUMN tx_type;
ALTER TABLE tokenpool DROP COLUMN tx_id;

DROP INDEX IF EXISTS tokenpool_fortx;

COMMIT;

BEGIN;
ALTER TABLE tokentransfer DROP COLUMN blockchain_event;
COMMIT;

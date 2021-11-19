BEGIN;
ALTER TABLE tokentransfer DROP COLUMN message_id;
COMMIT;

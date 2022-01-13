BEGIN;
ALTER TABLE tokentransfer DROP COLUMN tx_type;
ALTER TABLE tokentransfer DROP COLUMN tx_id;
ALTER TABLE tokentransfer ADD COLUMN blockchain_event UUID;
COMMIT;

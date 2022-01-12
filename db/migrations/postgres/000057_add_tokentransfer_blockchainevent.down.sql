BEGIN;
ALTER TABLE tokentransfer DROP COLUMN blockchain_event;
ALTER TABLE tokentransfer ADD COLUMN tx_type VARCHAR(64);
ALTER TABLE tokentransfer ADD COLUMN tx_id UUID;
COMMIT;

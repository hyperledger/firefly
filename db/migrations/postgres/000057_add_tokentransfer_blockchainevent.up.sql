BEGIN;
ALTER TABLE tokentransfer ADD COLUMN blockchain_event UUID;
COMMIT;

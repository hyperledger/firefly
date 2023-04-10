BEGIN;
DROP INDEX tokenpool_networkname;
ALTER TABLE tokenpool DROP COLUMN published;
ALTER TABLE tokenpool DROP COLUMN networkName;
COMMIT;

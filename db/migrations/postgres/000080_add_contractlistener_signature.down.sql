BEGIN;
ALTER TABLE contractlisteners DROP COLUMN signature;
COMMIT;

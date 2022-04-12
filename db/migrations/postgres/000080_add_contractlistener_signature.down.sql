BEGIN;
DROP INDEX contractlisteners_signature;
ALTER TABLE contractlisteners DROP COLUMN signature;
COMMIT;

BEGIN;
ALTER TABLE contractlisteners DROP COLUMN state;
ALTER TABLE contractlisteners DROP COLUMN updated;
COMMIT;
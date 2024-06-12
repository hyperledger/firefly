BEGIN;
ALTER TABLE contractlisteners DROP COLUMN filters;
COMMIT;
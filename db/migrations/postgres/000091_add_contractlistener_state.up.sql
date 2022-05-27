BEGIN;
ALTER TABLE contractlisteners ADD COLUMN state TEXT;
ALTER TABLE contractlisteners ADD updated BIGINT;
COMMIT;
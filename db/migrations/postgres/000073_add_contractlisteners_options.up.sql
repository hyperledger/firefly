BEGIN;
ALTER TABLE contractlisteners ADD COLUMN options TEXT;
COMMIT;
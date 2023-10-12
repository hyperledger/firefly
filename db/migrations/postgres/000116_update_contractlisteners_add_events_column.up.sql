BEGIN;
ALTER TABLE contractlisteners ADD COLUMN filters TEXT;
ALTER TABLE contractlisteners ALTER COLUMN event DROP NOT NULL;
ALTER TABLE contractlisteners ALTER COLUMN location DROP NOT NULL;
COMMIT:
BEGIN;
ALTER TABLE contractlisteners ADD COLUMN filters TEXT;
ALTER TABLE contractlisteners ALTER COLUMN event DROP NOT NULL;
ALTER TABLE contractlisteners ALTER COLUMN location DROP NOT NULL;
ALTER TABLE contractlisteners ADD COLUMN filter_hash CHAR(64);
CREATE INDEX contractlisteners_filter_hash ON contractlisteners(filter_hash);
COMMIT:
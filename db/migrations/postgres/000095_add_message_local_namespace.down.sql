BEGIN;
ALTER TABLE messages DROP COLUMN namespace_local;
ALTER TABLE groups DROP COLUMN namespace_local;
COMMIT;

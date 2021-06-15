BEGIN;
ALTER TABLE messages DROP COLUMN pending;
ALTER TABLE messages DROP COLUMN rejected;
CREATE INDEX messages_created ON messages(created);
COMMIT;

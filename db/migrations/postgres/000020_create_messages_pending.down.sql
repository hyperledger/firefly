BEGIN;
ALTER TABLE messages DROP COLUMN pending;
ALTER TABLE messages DROP COLUMN rejected;
CREATE INDEX messages_created ON messages(created);
CREATE INDEX messages_confirmed ON messages(confirmed);
COMMIT;

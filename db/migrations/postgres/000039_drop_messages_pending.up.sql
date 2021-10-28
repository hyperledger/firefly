BEGIN;
DROP INDEX messages_sortorder;
CREATE INDEX messages_sortorder ON messages(confirmed, created);

ALTER TABLE messages DROP COLUMN pending;
COMMIT;

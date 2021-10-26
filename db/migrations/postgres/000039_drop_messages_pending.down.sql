BEGIN;
ALTER TABLE messages ADD COLUMN pending SMALLINT;

DROP INDEX messages_sortorder;
CREATE INDEX messages_sortorder ON messages(pending, confirmed, created);
COMMIT;

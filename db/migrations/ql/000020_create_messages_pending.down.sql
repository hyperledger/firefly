DROP INDEX messages_sortorder;
ALTER TABLE messages DROP COLUMN pending;
ALTER TABLE messages DROP COLUMN rejected;
CREATE INDEX messages_created ON messages(created);

DROP INDEX messages_created;

ALTER TABLE messages ADD pending int64;
UPDATE messages SET pending = 1 WHERE confirmed = 0;
UPDATE messages SET pending = 0 WHERE confirmed != 0;

ALTER TABLE messages ADD rejected bool;
UPDATE messages SET rejected = FALSE;

CREATE INDEX messages_sortorder ON messages(pending, confirmed, created);

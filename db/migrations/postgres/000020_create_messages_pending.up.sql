BEGIN;

DROP INDEX messages_confirmed;
DROP INDEX messages_created;

ALTER TABLE messages ADD pending SMALLINT;
UPDATE messages SET pending = 1 WHERE confirmed = 0;
UPDATE messages SET pending = 0 WHERE confirmed != 0;
ALTER TABLE messages ALTER COLUMN pending SET NOT NULL;

ALTER TABLE messages ADD rejected BOOLEAN;
UPDATE messages SET rejected = FALSE;
ALTER TABLE messages ALTER COLUMN rejected SET NOT NULL;

CREATE INDEX messages_sortorder ON messages(pending, confirmed, created);

COMMIT;

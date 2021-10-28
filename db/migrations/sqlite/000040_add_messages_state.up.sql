ALTER TABLE messages ADD COLUMN state VARCHAR(64);
UPDATE messages SET state="pending" WHERE confirmed IS NULL;
UPDATE messages SET state="confirmed" WHERE confirmed IS NOT NULL AND rejected=false;
UPDATE messages SET state="rejected" WHERE confirmed IS NOT NULL AND rejected=true;

ALTER TABLE messages DROP COLUMN local;
ALTER TABLE messages DROP COLUMN rejected;

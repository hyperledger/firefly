ALTER TABLE messages ADD COLUMN local BOOLEAN;
ALTER TABLE messages ADD COLUMN rejected BOOLEAN;
UPDATE messages SET rejected=true WHERE state="rejected";

ALTER TABLE messages DROP COLUMN state;

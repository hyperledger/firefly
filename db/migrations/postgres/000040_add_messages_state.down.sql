BEGIN;
ALTER TABLE messages ADD COLUMN local BOOLEAN;
ALTER TABLE messages ADD COLUMN rejected BOOLEAN;
UPDATE messages SET rejected=true WHERE state="rejected";

ALTER TABLE messages DROP COLUMN state;
ALTER TABLE messages ALTER COLUMN local SET NOT NULL;
ALTER TABLE messages ALTER COLUMN rejected SET NOT NULL;
COMMIT;

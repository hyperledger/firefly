BEGIN;
ALTER TABLE messages ADD COLUMN namespace_local VARCHAR(64);
UPDATE messages SET namespace_local = namespace;
ALTER TABLE messages ALTER COLUMN namespace_local SET NOT NULL;

ALTER TABLE groups ADD COLUMN namespace_local VARCHAR(64);
UPDATE groups SET namespace_local = namespace;
ALTER TABLE groups ALTER COLUMN namespace_local SET NOT NULL;
COMMIT;

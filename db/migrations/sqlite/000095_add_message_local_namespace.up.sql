ALTER TABLE messages ADD COLUMN namespace_local VARCHAR(64);
UPDATE messages SET namespace_local = namespace;

ALTER TABLE groups ADD COLUMN namespace_local VARCHAR(64);
UPDATE groups SET namespace_local = namespace;

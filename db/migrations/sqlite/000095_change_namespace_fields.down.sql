ALTER TABLE messages DROP COLUMN namespace_local;
ALTER TABLE groups DROP COLUMN namespace_local;

ALTER TABLE namespaces ADD COLUMN id UUID;
ALTER TABLE namespaces ADD COLUMN message_id UUID;
ALTER TABLE namespaces ADD COLUMN ntype VARCHAR(64);
ALTER TABLE namespaces DROP COLUMN remote_name;

ALTER TABLE messages DROP COLUMN namespace_local;
ALTER TABLE groups DROP COLUMN namespace_local;

ALTER TABLE namespaces ADD COLUMN id UUID;
ALTER TABLE namespaces ADD COLUMN message_id UUID;
ALTER TABLE namespaces ADD COLUMN ntype VARCHAR(64);
ALTER TABLE namespaces DROP COLUMN remote_name;

DROP INDEX transactions_id;
CREATE UNIQUE INDEX transactions_id ON transactions(id);
DROP INDEX operations_id;
CREATE UNIQUE INDEX operations_id ON operations(id);

ALTER TABLE messages ADD COLUMN namespace_local VARCHAR(64);
UPDATE messages SET namespace_local = namespace;

ALTER TABLE groups ADD COLUMN namespace_local VARCHAR(64);
UPDATE groups SET namespace_local = namespace;

DROP INDEX namespaces_id;
ALTER TABLE namespaces DROP COLUMN id;
ALTER TABLE namespaces DROP COLUMN message_id;
ALTER TABLE namespaces DROP COLUMN ntype;
ALTER TABLE namespaces ADD COLUMN remote_name VARCHAR(64);
UPDATE namespaces SET remote_name = name;

DROP INDEX transactions_id;
CREATE UNIQUE INDEX transactions_id ON transactions(namespace, id);

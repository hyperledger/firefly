BEGIN;
DROP INDEX namespaces_id;
ALTER TABLE namespaces DROP COLUMN id;
ALTER TABLE namespaces DROP COLUMN message_id;
ALTER TABLE namespaces DROP COLUMN ntype;
COMMIT;

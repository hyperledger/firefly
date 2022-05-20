BEGIN;
ALTER TABLE namespaces DROP COLUMN contract_index;
COMMIT;

BEGIN;
ALTER TABLE namespaces ADD COLUMN firefly_contracts TEXT;
COMMIT;

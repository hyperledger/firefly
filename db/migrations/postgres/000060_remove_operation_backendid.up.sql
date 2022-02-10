BEGIN;
DROP INDEX operations_backend;
ALTER TABLE operations DROP COLUMN backend_id;
COMMIT;

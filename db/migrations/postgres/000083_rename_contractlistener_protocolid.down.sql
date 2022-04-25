BEGIN;
ALTER TABLE contractlisteners RENAME COLUMN backend_id TO protocol_id;
COMMIT;

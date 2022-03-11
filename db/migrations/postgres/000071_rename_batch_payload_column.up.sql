BEGIN;
ALTER TABLE batches RENAME COLUMN payload TO manifest;
COMMIT;

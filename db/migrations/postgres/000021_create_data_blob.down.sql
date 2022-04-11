BEGIN;
ALTER TABLE data DROP COLUMN blob_hash;
ALTER TABLE data DROP COLUMN blob_public;
DROP INDEX data_blobs;
COMMIT;

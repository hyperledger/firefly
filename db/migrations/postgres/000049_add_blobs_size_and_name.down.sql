BEGIN;
DROP INDEX data_blob_name;
DROP INDEX data_blob_size;

ALTER TABLE blobs DROP COLUMN size;
ALTER TABLE data DROP COLUMN blob_name;
ALTER TABLE data DROP COLUMN blob_size;
COMMIT;

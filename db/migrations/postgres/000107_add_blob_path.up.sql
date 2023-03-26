BEGIN;
ALTER TABLE data ADD COLUMN blob_path VARCHAR(1024);
CREATE INDEX data_blob_path ON data (blob_path);
UPDATE data SET blob_path = '';
ALTER TABLE data ALTER COLUMN blob_path SET NOT NULL;
COMMIT;
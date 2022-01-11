BEGIN;
ALTER TABLE blobs ADD COLUMN size BIGINT;

ALTER TABLE data ADD COLUMN blob_name VARCHAR(1024);
ALTER TABLE data ADD COLUMN blob_size BIGINT;

UPDATE blobs SET size = 0;
UPDATE data SET blob_size = 0, blob_name = '';

CREATE INDEX data_blob_name ON data(blob_name);
CREATE INDEX data_blob_size ON data(blob_size);

ALTER TABLE data ALTER COLUMN blob_name SET NOT NULL;
ALTER TABLE data ALTER COLUMN blob_size SET NOT NULL;

COMMIT;

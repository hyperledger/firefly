ALTER TABLE blobs ADD size BIGINT;

ALTER TABLE data ADD blob_name VARCHAR(1024);
ALTER TABLE data ADD blob_size BIGINT;
ALTER TABLE data ADD COLUMN value_size BIGINT;

UPDATE blobs SET size = 0;
UPDATE data SET blob_size = 0, blob_name = '', value_size = 0;

CREATE INDEX data_blob_name ON data(blob_name);
CREATE INDEX data_blob_size ON data(blob_size);

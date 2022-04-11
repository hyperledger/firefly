BEGIN;

ALTER TABLE data DROP COLUMN blobstore;
ALTER TABLE data ADD blob_hash CHAR(64);
ALTER TABLE data ADD blob_public VARCHAR(1024);

CREATE INDEX data_blobs ON data(blob_hash);

COMMIT;

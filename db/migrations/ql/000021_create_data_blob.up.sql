ALTER TABLE data DROP COLUMN blobstore;
ALTER TABLE data ADD blob_hash string;

CREATE INDEX data_blobs ON data(blob_hash);

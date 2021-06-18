ALTER TABLE data DROP COLUMN blobstore;
ALTER TABLE data ADD blob_hash string;
ALTER TABLE data ADD blob_public string;

CREATE INDEX data_blobs ON data(blob_hash);

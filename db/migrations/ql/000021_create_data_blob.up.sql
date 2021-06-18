ALTER TABLE data DROP COLUMN blobstore;
ALTER TABLE data ADD blob_hash string;
ALTER TABLE data ADD public_ref string;

CREATE INDEX data_blobs ON data(blob_hash);

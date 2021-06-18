BEGIN;

ALTER TABLE data DROP COLUMN blobstore;
ALTER TABLE data ADD blob_hash CHAR(64);
ALTER TABLE data ADD public_ref CHAR(64);

CREATE INDEX data_blobs ON data(blob_hash);

COMMIT;

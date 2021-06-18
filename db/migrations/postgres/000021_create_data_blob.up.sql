BEGIN;

ALTER TABLE data DROP COLUMN blobstore;
ALTER TABLE data ADD blob_hash CHAR(64);
ALTER TABLE data ADD blob_public CHAR(64);

-- Make payload_ref larger and a varchar, to accomodate more flexible IDs for non-IPFS public storage plugins
ALTER TABLE batches ALTER COLUMN payload_ref TYPE VARCHAR(256) USING payload_ref;

CREATE INDEX data_blobs ON data(blob_hash);

COMMIT;

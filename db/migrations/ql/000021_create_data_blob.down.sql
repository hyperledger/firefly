ALTER TABLE data DROP COLUMN blob_hash;
ALTER TABLE data DROP COLUMN public_ref;
DROP INDEX data_blobs;

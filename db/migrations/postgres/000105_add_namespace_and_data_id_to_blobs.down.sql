BEGIN;

DROP INDEX blobs_namespace_data_id;
DROP INDEX blobs_payload_ref;
ALTER TABLE blobs DROP COLUMN namespace;
ALTER TABLE blobs DROP COLUMN data_id;
CREATE INDEX blob_hash ON blobs(hash);

COMMIT;
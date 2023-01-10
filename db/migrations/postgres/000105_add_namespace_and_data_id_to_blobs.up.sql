BEGIN;

CREATE TABLE temp_blobs (
  seq            SERIAL          PRIMARY KEY,
  namespace      VARCHAR(64)     NOT NULL,
  hash           CHAR(64)        NOT NULL,
  payload_ref    VARCHAR(1024)   NOT NULL,
  created        BIGINT          NOT NULL,
  peer           VARCHAR(256)    NOT NULL,
  size           BIGINT,
  data_id        UUID            NOT NULL
);
INSERT INTO temp_blobs (namespace, data_id, hash, payload_ref, created, peer, size)
  SELECT DISTINCT data.namespace, data.id, data.blob_hash, blobs.payload_ref, blobs.created, blobs.peer, blobs.size
  FROM data
  LEFT JOIN blobs ON blobs.hash = data.blob_hash
  WHERE data.blob_hash IS NOT NULL
;
DROP INDEX blobs_hash;
DROP TABLE blobs;
ALTER TABLE temp_blobs RENAME TO blobs; 
CREATE INDEX blobs_namespace_data_id ON blobs(namespace, data_id);
CREATE INDEX blobs_payload_ref ON blobs(payload_ref);

COMMIT;
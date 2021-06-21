CREATE TABLE blobs (
  hash           string          NOT NULL,
  payload_ref    string          NOT NULL,
  created        int64           NOT NULL,
  peer           string          NOT NULL
);

CREATE INDEX blobs_hash ON blobs(hash);

CREATE TABLE blobs (
  seq            INTEGER         PRIMARY KEY AUTOINCREMENT,
  hash           CHAR(64)        NOT NULL,
  payload_ref    VARCHAR(1024)   NOT NULL,
  created        BIGINT          NOT NULL,
  peer           VARCHAR(256)    NOT NULL
);

CREATE INDEX blobs_hash ON blobs(hash);


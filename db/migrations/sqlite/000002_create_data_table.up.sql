CREATE TABLE data (
  seq              INTEGER         PRIMARY KEY AUTOINCREMENT,
  id               UUID            NOT NULL,
  validator        VARCHAR(64)     NOT NULL,
  namespace        VARCHAR(64)     NOT NULL,
  datatype_name    VARCHAR(64)     NOT NULL,
  datatype_version VARCHAR(64)     NOT NULL,
  hash             CHAR(64)        NOT NULL,
  created          BIGINT          NOT NULL,
  value            TEXT            NOT NULL,
  blob_hash        CHAR(64),
  blob_public      VARCHAR(1024)
);
CREATE UNIQUE INDEX data_id ON data(id);
CREATE INDEX data_hash ON data(namespace,hash);
CREATE INDEX data_created ON data(namespace,created);
CREATE INDEX data_blobs ON data(blob_hash);

BEGIN;
CREATE TABLE data (
  seq              SERIAL          PRIMARY KEY,
  id               UUID            NOT NULL,
  validator        VARCHAR(64)     NOT NULL,
  namespace        VARCHAR(64)     NOT NULL,
  datatype_name    VARCHAR(64)     NOT NULL,
  datatype_version VARCHAR(64)     NOT NULL,
  hash             CHAR(64)        NOT NULL,
  created          BIGINT          NOT NULL,
  value            TEXT            NOT NULL,
  blobstore        BOOLEAN         NOT NULL
);
CREATE UNIQUE INDEX data_id ON data(id);
CREATE INDEX data_hash ON data(namespace,hash);
CREATE INDEX data_created ON data(namespace,created);
COMMIT;
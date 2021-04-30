CREATE TABLE data (
  id          CHAR(36)        NOT NULL PRIMARY KEY,
  dtype       VARCHAR(64)     NOT NULL,
  namespace   VARCHAR(64)     NOT NULL,
  hash        CHAR(32)        NOT NULL,
  created     BIGINT          NOT NULL,
  value       BLOB            NOT NULL
);

CREATE INDEX data_search ON data(namespace,dtype,hash,created);
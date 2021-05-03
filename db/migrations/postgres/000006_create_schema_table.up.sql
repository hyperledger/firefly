CREATE TABLE schemas (
  id          CHAR(36)        NOT NULL PRIMARY KEY,
  stype       VARCHAR(64)     NOT NULL,
  namespace   VARCHAR(64)     NOT NULL,
  entity      VARCHAR(64)     NOT NULL,
  version     VARCHAR(64)     NOT NULL,
  hash        VARCHAR(32)     NOT NULL,
  created     INTEGER         NOT NULL,
  value       BLOB
);

CREATE UNIQUE INDEX schemas_unique ON schemas(namespace,entity,version);
CREATE INDEX schemas_search ON schemas(namespace,entity,version,created);
CREATE INDEX schemas_hash ON schemas(namespace,hash);

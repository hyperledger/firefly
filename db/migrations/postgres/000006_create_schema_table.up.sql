CREATE TABLE schemas (
  id          CHAR(36)        NOT NULL PRIMARY KEY,
  stype       VARCHAR(64)     NOT NULL,
  namespace   VARCHAR(64)     NOT NULL,
  entity      VARCHAR(64)     NOT NULL,
  version     VARCHAR(64)     NOT NULL,
  hash        CHAR(64)        NOT NULL,
  created     BIGINT          NOT NULL,
  value       JSONB
);

CREATE UNIQUE INDEX schemas_unique ON schemas(namespace,entity,version);
CREATE INDEX schemas_search ON schemas(namespace,entity,version,created);
CREATE INDEX schemas_hash ON schemas(namespace,hash);

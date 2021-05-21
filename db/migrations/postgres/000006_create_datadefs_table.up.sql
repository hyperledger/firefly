BEGIN;
CREATE TABLE datadefs (
  seq         SERIAL          PRIMARY KEY,
  id          UUID            NOT NULL,
  validator   VARCHAR(64)     NOT NULL,
  namespace   VARCHAR(64)     NOT NULL,
  name        VARCHAR(64)     NOT NULL,
  version     VARCHAR(64)     NOT NULL,
  hash        CHAR(64)        NOT NULL,
  created     BIGINT          NOT NULL,
  value       JSONB
);

CREATE UNIQUE INDEX datadefs_id ON data(id);
CREATE UNIQUE INDEX datadefs_unique ON datadefs(namespace,name,version);
CREATE INDEX datadefs_created ON datadefs(created);
COMMIT;
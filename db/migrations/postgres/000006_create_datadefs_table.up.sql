BEGIN;
CREATE SEQUENCE datadefs_seq;
CREATE TABLE datadefs (
  id          CHAR(36)        NOT NULL PRIMARY KEY,
  seq         BIGINT          NOT NULL DEFAULT nextval('datadefs_seq'),
  validator   VARCHAR(64)     NOT NULL,
  namespace   VARCHAR(64)     NOT NULL,
  name        VARCHAR(64)     NOT NULL,
  version     VARCHAR(64)     NOT NULL,
  hash        CHAR(64)        NOT NULL,
  created     BIGINT          NOT NULL,
  value       JSONB
);

CREATE UNIQUE INDEX datadefs_unique ON datadefs(namespace,name,version);
CREATE INDEX datadefs_created ON datadefs(created);
COMMIT;
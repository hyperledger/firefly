CREATE TABLE datatypes (
  seq         INTEGER         PRIMARY KEY AUTOINCREMENT,
  id          UUID            NOT NULL,
  message_id  UUID            NOT NULL,
  validator   VARCHAR(64)     NOT NULL,
  namespace   VARCHAR(64)     NOT NULL,
  name        VARCHAR(64)     NOT NULL,
  version     VARCHAR(64)     NOT NULL,
  hash        CHAR(64)        NOT NULL,
  created     BIGINT          NOT NULL,
  value       BYTEA
);

CREATE UNIQUE INDEX datatypes_id ON data(id);
CREATE UNIQUE INDEX datatypes_unique ON datatypes(namespace,name,version);
CREATE INDEX datatypes_created ON datatypes(created);

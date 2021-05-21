BEGIN;
CREATE SEQUENCE namespaces_seq;
CREATE TABLE namespaces (
  seq         SERIAL          PRIMARY KEY,
  id          UUID            NOT NULL,
  name        VARCHAR(64)     NOT NULL,
  ntype       VARCHAR(64)     NOT NULL,
  description VARCHAR(4096),
  created     BIGINT          NOT NULL,
  confirmed   BIGINT
);

CREATE UNIQUE INDEX namespaces_id ON operations(id);
CREATE UNIQUE INDEX namespaces_name ON namespaces(name);

COMMIT;
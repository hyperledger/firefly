BEGIN;
CREATE TABLE batches (
  seq         SERIAL          PRIMARY KEY,
  id          UUID            NOT NULL,
  btype       VARCHAR(64)     NOT NULL,
  namespace   VARCHAR(64)     NOT NULL,
  author      VARCHAR(1024)   NOT NULL,
  group_hash  CHAR(64),
  hash        CHAR(64),
  created     BIGINT          NOT NULL,
  payload     TEXT           NOT NULL,
  payload_ref CHAR(64),
  confirmed   BIGINT,
  tx_type     VARCHAR(64)     NOT NULL,
  tx_id       UUID
);

CREATE UNIQUE INDEX batches_id ON batches(id);
CREATE INDEX batches_created ON batches(namespace,created);
CREATE INDEX batches_fortx ON batches(namespace,tx_id);
COMMIT;
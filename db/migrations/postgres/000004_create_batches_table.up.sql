BEGIN;
CREATE SEQUENCE batches_seq;
CREATE TABLE batches (
  id          CHAR(36)        NOT NULL PRIMARY KEY,
  seq         BIGINT          NOT NULL DEFAULT nextval('batches_seq'),
  btype       VARCHAR(64)     NOT NULL,
  namespace   VARCHAR(64)     NOT NULL,
  author      VARCHAR(1024)   NOT NULL,
  hash        CHAR(64),
  created     BIGINT          NOT NULL,
  payload     JSONB           NOT NULL,
  payload_ref CHAR(64),
  confirmed   BIGINT,
  tx_type     VARCHAR(64)     NOT NULL,
  tx_id       CHAR(36)
);

CREATE INDEX batches_search ON batches(namespace,btype,author,confirmed,created);
CREATE INDEX batches_fortx ON batches(namespace,tx_id);
COMMIT;
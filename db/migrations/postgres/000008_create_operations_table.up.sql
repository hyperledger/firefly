BEGIN;
CREATE SEQUENCE operations_seq;
CREATE TABLE operations (
  id          UUID            NOT NULL PRIMARY KEY,
  seq         BIGINT          NOT NULL DEFAULT nextval('operations_seq'),
  namespace   VARCHAR(64)     NOT NULL,
  msg_id      UUID            NOT NULL,
  data_id     UUID,
  optype      VARCHAR(64)     NOT NULL,
  opstatus    VARCHAR(64)     NOT NULL,
  recipient   VARCHAR(1024),
  plugin      VARCHAR(64)     NOT NULL,
  backend_id  VARCHAR(256)    NOT NULL,
  created     BIGINT          NOT NULL,
  updated     BIGINT,
  error       VARCHAR         NOT NULL
);

CREATE UNIQUE INDEX operations_sequence ON operations(seq);
CREATE INDEX operations_created ON operations(created);
CREATE INDEX operations_backend ON operations(backend_id);

COMMIT;
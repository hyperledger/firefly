BEGIN;
CREATE SEQUENCE operations_seq;
CREATE TABLE operations (
  id          CHAR(36)        NOT NULL PRIMARY KEY,
  seq         BIGINT          NOT NULL DEFAULT nextval('operations_seq'),
  namespace   VARCHAR(64)     NOT NULL,
  msg_id      CHAR(36)        NOT NULL,
  data_id     CHAR(36),
  optype      VARCHAR(64)     NOT NULL,
  opstatus    VARCHAR(64)     NOT NULL,
  recipient   VARCHAR(1024),
  plugin      VARCHAR(64)     NOT NULL,
  backend_id  VARCHAR(256)    NOT NULL,
  created     BIGINT          NOT NULL,
  updated     BIGINT,
  error       VARCHAR         NOT NULL
);

CREATE INDEX operations_search ON operations(namespace,msg_id,optype,opstatus,error);
CREATE INDEX operations_backend ON operations(backend_id);

COMMIT;
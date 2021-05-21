BEGIN;
CREATE SEQUENCE transactions_seq;
CREATE TABLE transactions (
  id          UUID            NOT NULL PRIMARY KEY,
  seq         BIGINT          NOT NULL DEFAULT nextval('transactions_seq'),
  ttype       VARCHAR(64)     NOT NULL,
  namespace   VARCHAR(64)     NOT NULL,
  msg_id      UUID,
  batch_id    UUID,
  author      VARCHAR(1024)   NOT NULL,
  hash        CHAR(64)        NOT NULL,
  created     BIGINT          NOT NULL,
  protocol_id VARCHAR(256),
  status      VARCHAR(64)     NOT NULL,
  confirmed   BIGINT,
  info        JSONB
);

CREATE UNIQUE INDEX transactions_sequence ON data(seq);
CREATE INDEX transactions_created ON transactions(created);
CREATE INDEX transactions_protocol_id ON transactions(protocol_id);
COMMIT;
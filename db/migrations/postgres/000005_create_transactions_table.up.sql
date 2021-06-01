BEGIN;
CREATE TABLE transactions (
  seq         SERIAL          PRIMARY KEY,
  id          UUID            NOT NULL,
  ttype       VARCHAR(64)     NOT NULL,
  namespace   VARCHAR(64)     NOT NULL,
  ref         UUID,
  author      VARCHAR(1024)   NOT NULL,
  hash        CHAR(64)        NOT NULL,
  created     BIGINT          NOT NULL,
  protocol_id VARCHAR(256),
  status      VARCHAR(64)     NOT NULL,
  info        JSONB
);

CREATE UNIQUE INDEX transactions_id ON data(id);
CREATE INDEX transactions_created ON transactions(created);
CREATE INDEX transactions_protocol_id ON transactions(protocol_id);
CREATE INDEX transactions_ref ON transactions(ref);
COMMIT;
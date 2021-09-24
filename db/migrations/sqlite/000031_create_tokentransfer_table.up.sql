CREATE TABLE tokentransfer (
  seq              INTEGER         PRIMARY KEY AUTOINCREMENT,
  type             VARCHAR(64)     NOT NULL,
  pool_protocol_id VARCHAR(1024)   NOT NULL,
  token_index      VARCHAR(1024)   NOT NULL,
  key              VARCHAR(1024)   NOT NULL,
  from_key         VARCHAR(1024)   NULL,
  to_key           VARCHAR(1024)   NULL,
  amount           BIGINT          NOT NULL,
  protocol_id      VARCHAR(1024)   NOT NULL,
  created          BIGINT          NOT NULL
);

CREATE INDEX tokentransfer_pool ON tokentransfer(pool_protocol_id,token_index);
CREATE UNIQUE INDEX tokentransfer_protocolid ON tokentransfer(protocol_id);

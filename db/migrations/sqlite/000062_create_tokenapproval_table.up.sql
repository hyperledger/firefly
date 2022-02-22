CREATE TABLE tokenapproval (
  seq              INTEGER         PRIMARY KEY AUTOINCREMENT,
  local_id         UUID            NOT NULL,
  pool_id          VARCHAR(1024)   NOT NULL,
  key              VARCHAR(1024)   NOT NULL,
  operator_key     VARCHAR(1024)   NOT NULL,
  approved         BOOLEAN         NOT NULL,
  protocol_id      VARCHAR(1024)   NOT NULL,
  tx_type          VARCHAR(64),
  connector        VARCHAR(64),
  namespace        VARCHAR(64),
  info             TEXT,
  tx_id            UUID,
  blockchain_event UUID,
  created          BIGINT          NOT NULL
);

CREATE UNIQUE INDEX tokenapproval_id ON tokenapproval(local_id);
CREATE UNIQUE INDEX tokenapproval_protocolid ON tokenapproval(pool_id, protocol_id);
BEGIN;
CREATE TABLE blockchainevents (
  seq              SERIAL          PRIMARY KEY,
  id               UUID            NOT NULL,
  source           VARCHAR(256)    NOT NULL,
  namespace        VARCHAR(64)     NOT NULL,
  name             VARCHAR(256)    NOT NULL,
  protocol_id      VARCHAR(256)    NOT NULL,
  timestamp        BIGINT          NOT NULL,
  subscription_id  UUID,
  output           BYTEA,
  info             BYTEA,
  tx_type          VARCHAR(64),
  tx_id            UUID
);

CREATE INDEX blockchainevents_id ON blockchainevents(id);
CREATE INDEX blockchainevents_tx ON blockchainevents(tx_id);
CREATE INDEX blockchainevents_timestamp ON blockchainevents(timestamp);
CREATE INDEX blockchainevents_subscription_id ON blockchainevents(subscription_id);
COMMIT;

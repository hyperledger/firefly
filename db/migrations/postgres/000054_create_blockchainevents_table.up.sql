BEGIN;
CREATE TABLE blockchainevents (
  seq              INTEGER         PRIMARY KEY AUTOINCREMENT,
  id               UUID            NOT NULL,
  source           VARCHAR(1024)   NOT NULL,
  namespace        VARCHAR(64)     NOT NULL,
  name             VARCHAR(1024)   NOT NULL,
  subscription_id  UUID,
  outputs          BYTEA,
  info             BYTEA,
  timestamp        BIGINT          NOT NULL
);
COMMIT;

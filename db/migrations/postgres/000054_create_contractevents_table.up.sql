BEGIN;
CREATE TABLE contractevents (
  seq              SERIAL          PRIMARY KEY,
  id               UUID            NOT NULL,
  namespace        VARCHAR(64)     NOT NULL,
  name             VARCHAR(1024)   NOT NULL,
  subscription_id  UUID            NOT NULL,
  outputs          BYTEA,
  info             BYTEA,
  timestamp        BIGINT          NOT NULL
);
COMMIT;
BEGIN;
CREATE TABLE parked (
  seq            SERIAL          PRIMARY KEY,
  hash           CHAR(64)        NOT NULL,
  ledger_id      UUID,
  batch_id       UUID            NOT NULL,
  created        BIGINT          NOT NULL
);

CREATE INDEX parked_hash ON parked(hash);

COMMIT;
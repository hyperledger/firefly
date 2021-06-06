BEGIN;
CREATE TABLE parked (
  seq            SERIAL          PRIMARY KEY,
  pin            CHAR(64)        NOT NULL,
  ledger_id      UUID,
  batch_id       UUID            NOT NULL,
  created        BIGINT          NOT NULL
);

CREATE INDEX parked_hash ON parked(pin);
CREATE INDEX parked_batch ON parked(batch_id);

COMMIT;
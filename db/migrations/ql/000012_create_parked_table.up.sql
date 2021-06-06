CREATE TABLE parked (
  pin            string           NOT NULL,
  ledger_id      string,
  batch_id       string           NOT NULL,
  created        int64            NOT NULL
);

CREATE INDEX parked_hash ON parked(pin);
CREATE INDEX parked_batch ON parked(batch_id);

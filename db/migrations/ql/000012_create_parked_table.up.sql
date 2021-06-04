CREATE TABLE parked (
  hash           string           NOT NULL,
  ledger_id      string,
  batch_id       string           NOT NULL,
  created        int64            NOT NULL
);

CREATE INDEX parked_hash ON parked(hash);

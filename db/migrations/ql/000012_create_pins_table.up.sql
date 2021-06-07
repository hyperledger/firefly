CREATE TABLE pins (
  masked         bool             NOT NULL,
  hash           string           NOT NULL,
  batch_id       string           NOT NULL,
  idx            int64            NOT NUlL,
  dispatched     bool             NOT Null,
  created        int64            NOT NULL
);

CREATE UNIQUE INDEX pins_pin ON pins(hash, batch_id, idx);

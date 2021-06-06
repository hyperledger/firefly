CREATE TABLE pins (
  masked         bool             NOT NULL,
  hash           string           NOT NULL,
  batch_id       string           NOT NULL,
  created        int64            NOT NULL
);

CREATE UNIQUE INDEX pins_pin ON pins(hash, batch_id);

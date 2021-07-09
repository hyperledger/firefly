CREATE TABLE pins (
  seq            INTEGER         PRIMARY KEY AUTOINCREMENT,
  masked         BOOLEAN         NOT NULL,
  hash           CHAR(64)        NOT NULL,
  batch_id       UUID            NOT NULL,
  idx            BIGINT          NOT NULL,
  dispatched     BOOLEAN         NOT NULL,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX pins_pin ON pins(hash, batch_id, idx);
CREATE INDEX pins_dispatched ON pins(dispatched);


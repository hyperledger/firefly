CREATE TABLE nextpins (
  seq            INTEGER         PRIMARY KEY AUTOINCREMENT,
  context        CHAR(64)        NOT NULL,
  identity       VARCHAR(1024)   NOT NULL,
  hash           CHAR(64)        NOT NULL,
  nonce          BIGINT          NOT NULL
);

CREATE INDEX nextpins_hash ON nextpins(hash);


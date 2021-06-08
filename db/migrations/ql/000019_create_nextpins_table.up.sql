CREATE TABLE nextpins (
  context        string          NOT NULL,
  identity       string          NOT NULL,
  hash           string          NOT NULL,
  nonce          int64           NOT NULL
);

CREATE INDEX nextpins_hash ON nextpins(hash);

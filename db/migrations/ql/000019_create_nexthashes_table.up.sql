CREATE TABLE nexthashes (
  context        string          NOT NULL,
  identity       string          NOT NULL,
  hash           string          NOT NULL,
  nonce          int64           NOT NULL
);

CREATE INDEX nexthashes_hash ON nexthashes(hash);

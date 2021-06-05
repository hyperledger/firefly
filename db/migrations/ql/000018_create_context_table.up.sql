CREATE TABLE contexts (
  hash           string          NOT NULL,
  nonce          int64           NOT NULL,
  group_id       string          NOT NULL,
  topic          string          NOT NULL
);

CREATE INDEX contexts_hash ON contexts(hash);

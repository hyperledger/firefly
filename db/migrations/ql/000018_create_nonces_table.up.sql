CREATE TABLE nonces (
  context        string          NOT NULL,
  nonce          int64           NOT NULL,
  group_hash     string          NOT NULL,
  topic          string          NOT NULL
);

CREATE INDEX nonces_context ON nonces(context);
CREATE INDEX nonces_group ON nonces(group_hash);

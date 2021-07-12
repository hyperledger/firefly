CREATE TABLE nonces (
  seq            INTEGER         PRIMARY KEY AUTOINCREMENT,
  context        CHAR(64)        NOT NULL,
  nonce          BIGINT          NOT NULL,
  group_hash     CHAR(64)        NOT NULL,
  topic          VARCHAR(64)     NOT NULL
);

CREATE INDEX nonces_context ON nonces(context);
CREATE INDEX nonces_group ON nonces(group_hash);


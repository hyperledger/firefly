CREATE TABLE tokenaccount (
  seq            INTEGER         PRIMARY KEY AUTOINCREMENT,
  namespace      VARCHAR(64)     NOT NULL,
  pool_id        UUID            NOT NULL,
  token_index    VARCHAR(1024)   NOT NULL,
  identity       VARCHAR(1024)   NOT NULL,
  balance        BIGINT          DEFAULT 0,
  hash           CHAR(64)        NOT NULL
);

CREATE UNIQUE INDEX tokenaccount_hash ON tokenaccount(hash);
CREATE INDEX tokenaccount_pool ON tokenaccount(pool_id);

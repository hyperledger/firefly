DROP TABLE IF EXISTS tokenaccount;

CREATE TABLE tokenaccount (
  seq              INTEGER         PRIMARY KEY AUTOINCREMENT,
  pool_protocol_id VARCHAR(1024)   NOT NULL,
  token_index      VARCHAR(1024),
  identity         VARCHAR(1024)   NOT NULL,
  balance          VARCHAR(65)
);

CREATE UNIQUE INDEX tokenaccount_pool ON tokenaccount(identity,pool_protocol_id,token_index);

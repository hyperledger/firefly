BEGIN;
DROP TABLE IF EXISTS tokenaccount;

CREATE TABLE tokenaccount (
  seq              SERIAL          PRIMARY KEY,
  pool_protocol_id VARCHAR(1024)   NOT NULL,
  token_index      VARCHAR(1024),
  identity         VARCHAR(1024)   NOT NULL,
  balance          VARCHAR(65)
);

CREATE UNIQUE INDEX tokenaccount_pool ON tokenaccount(pool_protocol_id,token_index,identity);

COMMIT;

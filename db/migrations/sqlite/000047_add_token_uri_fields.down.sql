DROP INDEX tokenbalance_pool;
DROP INDEX tokenbalance_uri;

ALTER TABLE tokenbalance DROP COLUMN uri;
ALTER TABLE tokentransfer DROP COLUMN uri;

CREATE UNIQUE INDEX tokenbalance_pool ON tokenbalance(key,pool_id,token_index);

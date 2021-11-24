DROP INDEX tokenbalance_pool;

ALTER TABLE tokenbalance ADD COLUMN uri VARCHAR(1024);
ALTER TABLE tokentransfer ADD COLUMN uri VARCHAR(1024);

CREATE UNIQUE INDEX tokenbalance_pool ON tokenbalance(namespace,key,pool_id,token_index);
CREATE UNIQUE INDEX tokenbalance_uri ON tokenbalance(namespace,key,pool_id,uri);

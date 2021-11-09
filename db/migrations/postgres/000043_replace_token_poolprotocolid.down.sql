BEGIN;
DROP INDEX tokenbalance_pool;
DROP INDEX tokentransfer_pool;

ALTER TABLE tokenbalance ADD COLUMN pool_protocol_id VARCHAR(1024);
ALTER TABLE tokentransfer ADD COLUMN pool_protocol_id VARCHAR(1024);

UPDATE tokenbalance SET pool_protocol_id = pool.protocol_id
  FROM (SELECT protocol_id, id FROM tokenpool) AS pool
  WHERE tokenbalance.pool_id = pool.id;

UPDATE tokentransfer SET pool_protocol_id = pool.protocol_id
  FROM (SELECT protocol_id, id FROM tokenpool) AS pool
  WHERE tokentransfer.pool_id = pool.id;

ALTER TABLE tokenbalance DROP COLUMN pool_id;
ALTER TABLE tokentransfer DROP COLUMN pool_id;

ALTER TABLE tokenbalance ALTER COLUMN pool_protocol_id SET NOT NULL;
ALTER TABLE tokentransfer ALTER COLUMN pool_protocol_id SET NOT NULL;

CREATE UNIQUE INDEX tokenaccount_pool ON tokenbalance(key,pool_protocol_id,token_index);
CREATE INDEX tokentransfer_pool ON tokentransfer(pool_protocol_id,token_index);
COMMIT;

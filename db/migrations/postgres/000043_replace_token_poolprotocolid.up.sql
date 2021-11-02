BEGIN;
DROP INDEX tokenaccount_pool;
DROP INDEX tokentransfer_pool;

ALTER TABLE tokenbalance ADD COLUMN pool_id UUID;
ALTER TABLE tokentransfer ADD COLUMN pool_id UUID;

UPDATE tokenbalance SET pool_id = pool.id
  FROM (SELECT protocol_id, id FROM tokenpool) AS pool
  WHERE tokenbalance.pool_protocol_id = pool.protocol_id;

UPDATE tokentransfer SET pool_id = pool.id
  FROM (SELECT protocol_id, id FROM tokenpool) AS pool
  WHERE tokentransfer.pool_protocol_id = pool.protocol_id;

ALTER TABLE tokenbalance DROP COLUMN pool_protocol_id;
ALTER TABLE tokentransfer DROP COLUMN pool_protocol_id;

ALTER TABLE tokenbalance ALTER COLUMN pool_id SET NOT NULL;
ALTER TABLE tokentransfer ALTER COLUMN pool_id SET NOT NULL;

CREATE UNIQUE INDEX tokenbalance_pool ON tokenbalance(pool_id,token_index,key);
CREATE INDEX tokentransfer_pool ON tokentransfer(pool_id,token_index);
COMMIT;

ALTER TABLE tokenaccount ADD COLUMN namespace VARCHAR(64);
ALTER TABLE tokentransfer ADD COLUMN namespace VARCHAR(64);

UPDATE tokenaccount SET namespace = pool.namespace
  FROM (SELECT protocol_id, namespace FROM tokenpool) AS pool
  WHERE tokenaccount.pool_protocol_id = pool.protocol_id;

UPDATE tokentransfer SET namespace = pool.namespace
  FROM (SELECT protocol_id, namespace FROM tokenpool) AS pool
  WHERE tokentransfer.pool_protocol_id = pool.protocol_id;

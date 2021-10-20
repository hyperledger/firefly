BEGIN;

ALTER TABLE tokenaccount ADD COLUMN connector VARCHAR(64);
ALTER TABLE tokentransfer ADD COLUMN connector VARCHAR(64);

UPDATE tokenaccount SET connector = pool.connector
  FROM (SELECT protocol_id, connector FROM tokenpool) AS pool
  WHERE tokenaccount.pool_protocol_id = pool.protocol_id;

UPDATE tokentransfer SET connector = pool.connector
  FROM (SELECT protocol_id, connector FROM tokenpool) AS pool
  WHERE tokentransfer.pool_protocol_id = pool.protocol_id;

ALTER TABLE tokenaccount ALTER COLUMN connector SET NOT NULL;
ALTER TABLE tokentransfer ALTER COLUMN connector SET NOT NULL;

COMMIT;

BEGIN;
DROP INDEX tokentransfer_protocolid;
DROP INDEX tokenapproval_protocolid;

-- De-duplicate existing approvals by adding the pool seq number to the protocol_id
UPDATE tokenapproval
  SET protocol_id = (protocol_id || '/' || pool.seq)
  FROM (SELECT seq, id FROM tokenpool) AS pool
  WHERE length(protocol_id) = 26 AND CAST(pool.id AS TEXT) = pool_id;

CREATE UNIQUE INDEX tokentransfer_protocolid ON tokentransfer(connector, protocol_id);
CREATE UNIQUE INDEX tokenapproval_protocolid ON tokenapproval(connector, protocol_id);
COMMIT;

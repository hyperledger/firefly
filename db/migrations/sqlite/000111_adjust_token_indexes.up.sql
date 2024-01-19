DROP INDEX tokenpool_locator;
DROP INDEX tokentransfer_protocolid;
DROP INDEX tokenapproval_protocolid;

CREATE UNIQUE INDEX tokentransfer_protocolid ON tokentransfer(namespace, pool_id, protocol_id);
CREATE UNIQUE INDEX tokenapproval_protocolid ON tokenapproval(namespace, pool_id, protocol_id);

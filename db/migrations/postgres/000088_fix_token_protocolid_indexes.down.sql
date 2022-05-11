BEGIN;
DROP INDEX tokentransfer_protocolid;
DROP INDEX tokenapproval_protocolid;
CREATE UNIQUE INDEX tokentransfer_protocolid ON tokentransfer(pool_id, protocol_id);
CREATE UNIQUE INDEX tokenapproval_protocolid ON tokenapproval(pool_id, protocol_id);
COMMIT;

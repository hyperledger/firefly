DROP INDEX tokentransfer_protocolid;
DROP INDEX tokenapproval_protocolid;
CREATE UNIQUE INDEX tokentransfer_protocolid ON tokentransfer(connector, protocol_id);
CREATE UNIQUE INDEX tokenapproval_protocolid ON tokenapproval(connector, protocol_id);

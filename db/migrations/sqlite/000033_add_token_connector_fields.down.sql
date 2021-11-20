DROP INDEX tokentransfer_protocolid;
CREATE UNIQUE INDEX tokentransfer_protocolid ON tokentransfer(protocol_id);

ALTER TABLE tokenaccount DROP COLUMN connector;
ALTER TABLE tokentransfer DROP COLUMN connector;

DROP INDEX tokenapproval_subject;
DROP INDEX tokentransfer_protocolid;
DROP INDEX tokenpool_locator;

ALTER TABLE tokenapproval DROP COLUMN subject;
ALTER TABLE tokenpool RENAME COLUMN locator TO protocol_id;

CREATE UNIQUE INDEX tokenpool_protocolid ON tokenpool(connector, protocol_id);
CREATE UNIQUE INDEX tokentransfer_protocolid ON tokentransfer(connector, protocol_id);

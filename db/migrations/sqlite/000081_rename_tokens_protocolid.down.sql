DROP INDEX tokenapproval_subject;
DROP INDEX tokentransfer_subject;
DROP INDEX tokenpool_locator;
ALTER TABLE tokenapproval RENAME COLUMN subject TO protocol_id;
ALTER TABLE tokentransfer RENAME COLUMN subject TO protocol_id;
ALTER TABLE tokenpool RENAME COLUMN locator TO protocol_id;
CREATE UNIQUE INDEX tokenapproval_protocolid ON tokenapproval(pool_id, protocol_id);
CREATE UNIQUE INDEX tokentransfer_protocolid ON tokentransfer(connector, protocol_id);
CREATE UNIQUE INDEX tokenpool_protocolid ON tokenpool(connector, protocol_id);

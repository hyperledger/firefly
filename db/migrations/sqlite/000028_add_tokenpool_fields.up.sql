DELETE FROM tokenpool;
ALTER TABLE tokenpool ADD COLUMN connector VARCHAR(64) NOT NULL;
ALTER TABLE tokenpool ADD COLUMN symbol VARCHAR(64);
ALTER TABLE tokenpool ADD COLUMN message_id UUID;

DROP INDEX tokenpool_protocolid;
CREATE UNIQUE INDEX tokenpool_protocolid ON tokenpool(connector,protocol_id);

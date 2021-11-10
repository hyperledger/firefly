DROP INDEX tokenpool_protocolid;
CREATE UNIQUE INDEX tokenpool_protocolid ON tokenpool(protocol_id);

ALTER TABLE tokenpool DROP COLUMN connector;
ALTER TABLE tokenpool DROP COLUMN symbol;
ALTER TABLE tokenpool DROP COLUMN message_id;

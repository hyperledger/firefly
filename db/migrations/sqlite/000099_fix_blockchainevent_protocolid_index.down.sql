DROP INDEX blockchainevents_protocolid;
DROP INDEX blockchainevents_listener_protocolid;
CREATE UNIQUE INDEX blockchainevents_protocolid ON blockchainevents(namespace, listener_id, protocol_id);

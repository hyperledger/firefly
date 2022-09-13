BEGIN;
DROP INDEX blockchainevents_protocolid;
CREATE UNIQUE INDEX blockchainevents_protocolid ON blockchainevents(namespace, protocol_id) WHERE listener_id IS NULL;
CREATE UNIQUE INDEX blockchainevents_listener_protocolid ON blockchainevents(namespace, listener_id, protocol_id) WHERE listener_id IS NOT NULL;
COMMIT;

DROP INDEX blockchainevents_protocolid;
DELETE FROM blockchainevents WHERE listener_id IS NULL AND seq NOT IN (
  SELECT MIN(seq) FROM blockchainevents WHERE listener_id IS NULL GROUP BY namespace, protocol_id);
CREATE UNIQUE INDEX blockchainevents_protocolid ON blockchainevents(namespace, protocol_id) WHERE listener_id IS NULL;
CREATE UNIQUE INDEX blockchainevents_listener_protocolid ON blockchainevents(namespace, listener_id, protocol_id) WHERE listener_id IS NOT NULL;

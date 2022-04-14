DELETE FROM blockchainevents WHERE seq NOT IN (SELECT MIN(seq) FROM blockchainevents GROUP BY namespace, listener_id, protocol_id);
CREATE UNIQUE INDEX blockchainevents_protocolid ON blockchainevents(namespace, listener_id, protocol_id);

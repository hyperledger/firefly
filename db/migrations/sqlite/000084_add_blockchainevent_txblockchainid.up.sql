ALTER TABLE blockchainevents ADD COLUMN tx_blockchain_id VARCHAR(1024);
CREATE INDEX blockchainevents_txblockchainid ON blockchainevents(tx_blockchain_id);

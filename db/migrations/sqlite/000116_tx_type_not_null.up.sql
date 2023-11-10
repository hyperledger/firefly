UPDATE messages SET tx_parent_type = '' WHERE tx_parent_type IS NULL;
ALTER TABLE messages RENAME COLUMN tx_parent_type TO tx_parent_type_temp;
ALTER TABLE messages ADD COLUMN tx_parent_type VARCHAR(64) DEFAULT '' NOT NULL;
UPDATE messages SET tx_parent_type = tx_parent_type_temp;
ALTER TABLE messages DROP COLUMN tx_parent_type_temp;
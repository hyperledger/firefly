BEGIN;
ALTER TABLE messages ADD COLUMN tx_id UUID;
ALTER TABLE messages ADD COLUMN tx_parent_type VARCHAR(64);
ALTER TABLE messages ADD COLUMN tx_parent_id UUID;
COMMIT;

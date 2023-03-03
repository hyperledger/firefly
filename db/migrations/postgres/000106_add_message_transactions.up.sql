BEGIN;
ALTER TABLE messages ADD COLUMN tx_id UUID;
UPDATE messages SET tx_id = batches.tx_id
  FROM batches WHERE messages.batch_id = batches.id AND messages.tx_id IS NULL;
ALTER TABLE messages ADD COLUMN tx_parent_type VARCHAR(64);
ALTER TABLE messages ADD COLUMN tx_parent_id UUID;
COMMIT;

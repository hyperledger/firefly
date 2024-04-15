BEGIN;
UPDATE messages SET tx_parent_type = ''
  WHERE tx_parent_type IS NULL;
ALTER TABLE messages ALTER COLUMN tx_parent_type SET NOT NULL;
COMMIT;

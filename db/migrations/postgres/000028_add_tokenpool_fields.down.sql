BEGIN;
ALTER TABLE tokenpool DROP COLUMN connector;
ALTER TABLE tokenpool DROP COLUMN symbol;
ALTER TABLE tokenpool DROP COLUMN message_id;
COMMIT;

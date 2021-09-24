BEGIN;
ALTER TABLE tokenaccount RENAME COLUMN protocol_id TO pool_protocol_id;
COMMIT;

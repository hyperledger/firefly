BEGIN;

DROP INDEX tokenapproval_messageid;
ALTER TABLE tokenapproval DROP COLUMN message_id;
ALTER TABLE tokenapproval DROP COLUMN message_hash;

COMMIT;

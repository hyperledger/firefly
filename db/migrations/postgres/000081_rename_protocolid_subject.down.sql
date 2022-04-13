BEGIN;
DROP INDEX tokenapproval_subject;
DROP INDEX tokentransfer_subject;
ALTER TABLE tokenapproval RENAME COLUMN subject TO protocol_id;
ALTER TABLE tokentransfer RENAME COLUMN subject TO protocol_id;
CREATE UNIQUE INDEX tokenapproval_protocolid ON tokenapproval(pool_id, protocol_id);
CREATE UNIQUE INDEX tokentransfer_protocolid ON tokentransfer(protocol_id);
COMMIT;

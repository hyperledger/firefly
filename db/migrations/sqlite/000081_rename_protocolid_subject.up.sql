DROP INDEX tokenapproval_protocolid;
DROP INDEX tokentransfer_protocolid;
ALTER TABLE tokenapproval RENAME COLUMN protocol_id TO subject;
ALTER TABLE tokentransfer RENAME COLUMN protocol_id TO subject;
CREATE UNIQUE INDEX tokenapproval_subject ON tokenapproval(pool_id, subject);
CREATE UNIQUE INDEX tokentransfer_subject ON tokentransfer(pool_id, subject);

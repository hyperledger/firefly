DROP INDEX tokenapproval_protocolid;
DROP INDEX tokenapproval_subject;

ALTER TABLE tokenapproval RENAME COLUMN pool_id TO pool_id_old;
ALTER TABLE tokenapproval ADD COLUMN pool_id VARCHAR(1024);
UPDATE tokenapproval SET pool_id = CAST(pool_id_old AS VARCHAR);
ALTER TABLE tokenapproval DROP COLUMN pool_id_old;

CREATE UNIQUE INDEX tokenapproval_protocolid ON tokenapproval(pool_id, protocol_id);
CREATE INDEX tokenapproval_subject ON tokenapproval(pool_id, subject);

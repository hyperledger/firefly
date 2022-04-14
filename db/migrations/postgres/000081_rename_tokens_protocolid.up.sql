BEGIN;
DROP INDEX tokenapproval_protocolid;
DROP INDEX tokentransfer_protocolid;
DROP INDEX tokenpool_protocolid;

ALTER TABLE tokenapproval ADD COLUMN subject VARCHAR(1024);
UPDATE tokenapproval SET subject = protocol_id;
ALTER TABLE tokenapproval ALTER COLUMN subject SET NOT NULL;
ALTER TABLE tokenapproval ADD COLUMN active BOOLEAN;
UPDATE tokenapproval SET active = true;

ALTER TABLE tokenpool RENAME COLUMN protocol_id TO locator;

CREATE UNIQUE INDEX tokenapproval_protocolid ON tokenapproval(pool_id, protocol_id);
CREATE UNIQUE INDEX tokenapproval_subject ON tokenapproval(pool_id, subject);
CREATE UNIQUE INDEX tokentransfer_protocolid ON tokentransfer(pool_id, protocol_id);
CREATE UNIQUE INDEX tokenpool_locator ON tokenpool(connector, locator);
COMMIT;

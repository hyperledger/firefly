BEGIN;
ALTER TABLE tokenpool ADD COLUMN published BOOLEAN DEFAULT false;
UPDATE tokenpool SET published = true WHERE message_id IS NOT NULL;
ALTER TABLE tokenpool ADD COLUMN networkName VARCHAR(64) DEFAULT '';
UPDATE tokenpool SET networkName = name WHERE message_id IS NOT NULL;
CREATE UNIQUE INDEX tokenpool_networkname ON tokenpool(namespace,networkName);
COMMIT;

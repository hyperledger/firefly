BEGIN;
ALTER TABLE contractapis ADD COLUMN published BOOLEAN DEFAULT false;
UPDATE contractapis SET published = true WHERE message_id IS NOT NULL;
ALTER TABLE contractapis ADD COLUMN network_name VARCHAR(64);
UPDATE contractapis SET network_name = name WHERE message_id IS NOT NULL;
CREATE UNIQUE INDEX contractapis_networkname ON contractapis(namespace,network_name);
COMMIT;

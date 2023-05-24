ALTER TABLE tokenpool ADD COLUMN published BOOLEAN DEFAULT false;
UPDATE tokenpool SET published = true WHERE message_id IS NOT NULL;

ALTER TABLE tokenpool ADD COLUMN network_name VARCHAR(64);
UPDATE tokenpool SET network_name = name WHERE message_id IS NOT NULL;

ALTER TABLE tokenpool ADD COLUMN plugin_data TEXT;
UPDATE tokenpool SET plugin_data = namespace;

CREATE UNIQUE INDEX tokenpool_networkname ON tokenpool(namespace,network_name);

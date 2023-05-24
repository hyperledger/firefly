ALTER TABLE ffi ADD COLUMN published BOOLEAN DEFAULT false;
UPDATE ffi SET published = true WHERE message_id IS NOT NULL;
ALTER TABLE ffi ADD COLUMN network_name VARCHAR(64);
UPDATE ffi SET network_name = name WHERE message_id IS NOT NULL;
CREATE UNIQUE INDEX ffi_networkname ON ffi(namespace,network_name,version);

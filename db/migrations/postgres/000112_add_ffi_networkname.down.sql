BEGIN;
DROP INDEX ffi_networkname;
ALTER TABLE ffi DROP COLUMN published;
ALTER TABLE ffi DROP COLUMN network_name;
COMMIT;

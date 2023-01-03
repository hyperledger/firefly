BEGIN;
ALTER TABLE tokenpool DROP COLUMN interface;
ALTER TABLE tokenpool DROP COLUMN interface_format;
COMMIT;

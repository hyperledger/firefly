BEGIN;
ALTER TABLE tokenpool ADD COLUMN interface UUID;
ALTER TABLE tokenpool ADD COLUMN interface_format VARCHAR(64) DEFAULT '';
ALTER TABLE tokenpool ADD COLUMN methods TEXT;
COMMIT;

BEGIN;
ALTER TABLE tokenpool ADD COLUMN info TEXT;
COMMIT;
BEGIN;
ALTER TABLE tokenaccount RENAME COLUMN key TO identity;
COMMIT;

BEGIN;
ALTER TABLE tokenaccount RENAME COLUMN identity TO key;
COMMIT;

BEGIN;
ALTER TABLE tokenaccount ADD COLUMN updated BIGINT;
UPDATE tokenaccount SET updated = 0;
ALTER TABLE tokenaccount ALTER COLUMN updated SET NOT NULL;
COMMIT;

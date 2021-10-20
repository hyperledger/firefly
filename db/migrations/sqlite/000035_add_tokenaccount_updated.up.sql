ALTER TABLE tokenaccount ADD COLUMN updated BIGINT;
UPDATE tokenaccount SET updated = 0;

ALTER TABLE data RENAME COLUMN value TO value_old;
ALTER TABLE data ADD COLUMN value TEXT;
UPDATE data SET value = value_old;
ALTER TABLE data DROP COLUMN value_old;

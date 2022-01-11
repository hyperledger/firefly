ALTER TABLE data ADD COLUMN value_size BIGINT;

UPDATE data SET value_size = 0;

ALTER TABLE data ADD COLUMN public VARCHAR(1024);
UPDATE data SET public = '';
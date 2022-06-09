ALTER TABLE pins ADD COLUMN namespace VARCHAR(64);
UPDATE pins SET namespace = "ff_system";

ALTER TABLE contractlisteners RENAME COLUMN location TO location_old;
ALTER TABLE contractlisteners ADD COLUMN location TEXT;
UPDATE contractlisteners SET location = location_old;
ALTER TABLE contractlisteners DROP COLUMN location_old;
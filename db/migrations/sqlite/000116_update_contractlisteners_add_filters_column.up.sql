ALTER TABLE contractlisteners ADD COLUMN filters TEXT;

-- SQLite doesn't support dropping NOT NULL constraints so we have to move
-- everything to a temp colum, create a new column, then copy the data back
ALTER TABLE contractlisteners RENAME COLUMN event TO event_tmp;
ALTER TABLE contractlisteners RENAME COLUMN location TO location_tmp;
ALTER TABLE contractlisteners ADD COLUMN event TEXT;
ALTER TABLE contractlisteners ADD LOCATION event TEXT;
UPDATE contractlisteners SET event = event_tmp;
UPDATE contractlisteners SET location = location_tmp;
ALTER TABLE contractlisteners DROP COLUMN event_tmp;
ALTER TABLE contractlisteners DROP COLUMN location_tmp;

ALTER TABLE contractlisteners ADD COLUMN filter_hash CHAR(64);
CREATE INDEX contractlisteners_filter_hash ON contractlisteners(filter_hash);

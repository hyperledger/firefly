BEGIN;
ALTER TABLE contractlisteners DROP COLUMN filters;
-- no down for the VARCHAR change
COMMIT;
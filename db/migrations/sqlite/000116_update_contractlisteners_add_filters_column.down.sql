ALTER TABLE contractlisteners DROP COLUMN filters;
ALTER TABLE contractlisteners DROP COLUMN filter_hash;
DROP INDEX contractlisteners_filter_hash;

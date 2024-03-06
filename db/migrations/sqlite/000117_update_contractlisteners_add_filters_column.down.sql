ALTER TABLE contractlisteners DROP COLUMN filters;
DROP INDEX contractlisteners_filter_hash;
ALTER TABLE contractlisteners DROP COLUMN filter_hash;

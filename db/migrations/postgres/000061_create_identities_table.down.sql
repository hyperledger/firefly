BEGIN;

DROP INDEX identities_id;
DROP INDEX identities_did;
DROP INDEX identities_name;

DROP TABLE IF EXISTS identities;

COMMIT;
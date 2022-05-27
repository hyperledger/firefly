BEGIN;
DROP INDEX identities_did;
CREATE UNIQUE INDEX identities_did ON identities(namespace, did);

DROP INDEX verifiers_value;
CREATE UNIQUE INDEX verifiers_value ON verifiers(namespace, vtype, value);
COMMIT;

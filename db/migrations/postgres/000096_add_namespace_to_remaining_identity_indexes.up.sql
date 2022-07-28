BEGIN;
DROP INDEX identities_id;
CREATE UNIQUE INDEX identities_id ON identities(namespace, id);

DROP INDEX verifiers_hash;
CREATE UNIQUE INDEX verifiers_hash on verifiers(namespace, hash);

DROP INDEX verifiers_identity;
CREATE UNIQUE INDEX verifiers_identity on verifiers(namespace, identity);
COMMIT;
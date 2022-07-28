BEGIN;
DROP INDEX identities_id;
CREATE UNIQUE INDEX identities_id ON identities(id);

DROP INDEX verifiers_hash;
CREATE UNIQUE INDEX verifiers_hash on verifiers(hash);

DROP INDEX verifiers_identity;
CREATE UNIQUE INDEX verifiers_identity on verifiers(identity);
COMMIT;
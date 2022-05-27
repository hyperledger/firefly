DROP INDEX identities_did;
CREATE UNIQUE INDEX identities_did ON identities(did);

DROP INDEX verifiers_value;
CREATE UNIQUE INDEX verifiers_value ON verifiers(vtype, value);

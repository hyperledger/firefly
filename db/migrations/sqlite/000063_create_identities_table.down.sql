CREATE TABLE orgs (
  seq            INTEGER         PRIMARY KEY AUTOINCREMENT,
  id             UUID            NOT NULL,
  message_id     UUID            NOT NULL,
  name           VARCHAR(64)     NOT NULL,
  parent         VARCHAR(1024),
  identity       VARCHAR(1024)   NOT NULL,
  description    VARCHAR(4096)   NOT NULL,
  profile        TEXT,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX orgs_id ON orgs(id);
CREATE UNIQUE INDEX orgs_identity ON orgs(identity);
CREATE UNIQUE INDEX orgs_name ON orgs(name);

CREATE TABLE nodes (
  seq            INTEGER         PRIMARY KEY AUTOINCREMENT,
  id             UUID            NOT NULL,  
  message_id     UUID            NOT NULL,
  owner          VARCHAR(1024)   NOT NULL,
  name           VARCHAR(64)     NOT NULL,
  description    VARCHAR(4096)   NOT NULL,
  dx_peer        VARCHAR(256),
  dx_endpoint    TEXT,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX nodes_id ON nodes(id);
CREATE UNIQUE INDEX nodes_owner ON nodes(owner,name);
CREATE UNIQUE INDEX nodes_peer ON nodes(dx_peer);

-- We only reconstitute orgs that were dropped during the original up migration.
-- These have the UUID of the verifier set to the same UUID as the org.
INSERT INTO orgs (
    id,
    parent,
    message_id,
    name,
    description,
    profile,
    created,
    identity
  ) SELECT 
    i.id,
    COALESCE(pv.value, '') as parent,
    i.messages_claim,
    i.name,
    i.description,
    i.profile,
    i.created,
    v.value as identity
  FROM identities as i
  LEFT JOIN verifiers v ON v.hash = REPLACE(i.id,'-','') || REPLACE(i.id,'-','')
  LEFT JOIN verifiers pv ON pv.hash = REPLACE(i.parent,'-','') || REPLACE(i.parent,'-','')
  WHERE i.did LIKE 'did:firefly:org/%' AND v.hash IS NOT NULL;

-- We only reconstitute nodes that were dropped during the original up migration.
-- These have the Hash of the verifier set to the bytes from the UUID of the node (by taking the string and removing the dashes).
INSERT INTO nodes (
    id,
    owner,
    message_id,
    name,
    description,
    dx_endpoint,
    created,
    dx_peer
  ) SELECT 
    i.id,
    COALESCE(pv.value, '') as owner,
    i.messages_claim,
    i.name,
    i.description,
    i.profile,
    i.created,
    v.value as dx_peer
  FROM identities as i
  LEFT JOIN verifiers v ON v.hash = REPLACE(i.id,'-','') || REPLACE(i.id,'-','')
  LEFT JOIN verifiers pv ON pv.hash = REPLACE(i.parent,'-','') || REPLACE(i.parent,'-','')
  WHERE i.did LIKE 'did:firefly:node/%' AND v.hash IS NOT NULL;

DROP INDEX identities_id;
DROP INDEX identities_did;
DROP INDEX identities_name;

DROP TABLE IF EXISTS identities;

DROP INDEX verifiers_hash;
DROP INDEX verifiers_value;
DROP INDEX verifiers_identity;

DROP TABLE IF EXISTS verifiers;

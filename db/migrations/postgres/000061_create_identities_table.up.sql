BEGIN;

CREATE TABLE identities (
  seq            SERIAL          PRIMARY KEY,
  id             UUID            NOT NULL,
  did            VARCHAR(256)    NOT NULL,
  parent         UUID,
  message_id     UUID            NOT NULL,
  itype          VARCHAR(64)     NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  name           VARCHAR(64)     NOT NULL,
  description    VARCHAR(4096)   NOT NULL,
  profile        TEXT,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX identities_id ON identities(id);
CREATE UNIQUE INDEX identities_did ON identities(did);
CREATE UNIQUE INDEX identities_name ON identities(itype, namespace, name);

CREATE TABLE verifiers (
  seq            SERIAL          PRIMARY KEY,
  id             UUID            NOT NULL,
  identity       UUID            NOT NULL,
  vtype          VARCHAR(256)    NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  value          TEXT            NOT NULL,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX verifiers_id ON verifiers(id);
CREATE UNIQUE INDEX verifiers_value ON verifiers(vtype, namespace, value);
CREATE UNIQUE INDEX verifiers_identity ON verifiers(identity);

INSERT INTO identities (
    id,
    did,
    parent,
    message_id,
    itype,
    namespace,
    name,
    description,
    profile,
    created
  ) SELECT 
    o1.id,
    'did:firefly:org/' || o1.name,
    o2.id,
    o1.message_id,
    'org',
    'ff_system',
    o1.name,
    o1.description,
    o1.profile,
    o1.created    
  FROM orgs as o1
  LEFT JOIN orgs o2 ON o2.identity = o1.parent;

INSERT INTO identities (
    id,
    did,
    parent,
    message_id,
    itype,
    namespace,
    name,
    description,
    profile,
    created
  ) SELECT 
    n.id,
    'did:firefly:node/' || n.name,
    o.id,
    n.message_id,
    'node',
    'ff_system',
    n.name,
    n.description,
    n.dx_endpoint,
    n.created    
  FROM nodes as n
  LEFT JOIN orgs o ON o.identity = n.owner;

INSERT INTO verifiers (
    id,
    identity,
    vtype,
    value,
    created
  ) SELECT 
    o.id,
    o.id,
    'EcdsaSecp256k1VerificationKey2019',
    o.identity,
    o.created    
  FROM orgs as o WHERE o.identity LIKE '0x%';

INSERT INTO verifiers (
    id,
    identity,
    vtype,
    value,
    created
  ) SELECT 
    o.id,
    o.id,
    'HyperledgerFabricMSPIdentity',
    o.identity,
    o.created
  FROM orgs as o WHERE o.identity NOT LIKE '0x%';

INSERT INTO verifiers (
    id,
    identity,
    vtype,
    value,
    created
  ) SELECT 
    n.id,
    n.id,
    'FireFlyDataExchangePeerId',
    n.dx_peer,
    n.created
  FROM nodes as n;

DROP TABLE orgs;
DROP TABLE nodes;

COMMIT;
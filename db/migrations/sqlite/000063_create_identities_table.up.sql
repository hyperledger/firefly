CREATE TABLE identities (
  seq                   INTEGER         PRIMARY KEY AUTOINCREMENT,
  id                    UUID            NOT NULL,
  did                   VARCHAR(256)    NOT NULL,
  parent                UUID,
  messages_claim        UUID            NOT NULL,
  messages_verification UUID,
  messages_update       UUID,
  itype                 VARCHAR(64)     NOT NULL,
  namespace             VARCHAR(64)     NOT NULL,
  name                  VARCHAR(64)     NOT NULL,
  description           VARCHAR(4096)   NOT NULL,
  profile               TEXT,
  created               BIGINT          NOT NULL,
  updated               BIGINT          NOT NULL
);

CREATE UNIQUE INDEX identities_id ON identities(id);
CREATE UNIQUE INDEX identities_did ON identities(did);
CREATE UNIQUE INDEX identities_name ON identities(itype, namespace, name);

CREATE TABLE verifiers (
  seq            INTEGER         PRIMARY KEY AUTOINCREMENT,
  id             UUID            NOT NULL,
  identity       UUID            NOT NULL,
  vtype          VARCHAR(256)    NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  value          TEXT            NOT NULL,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX verifiers_id ON verifiers(id);
CREATE UNIQUE INDEX verifiers_value ON verifiers(vtype, value);
CREATE UNIQUE INDEX verifiers_identity ON verifiers(identity);

INSERT INTO identities (
    id,
    did,
    parent,
    messages_claim,
    namespace,
    name,
    description,
    profile,
    created,
    updated
  ) SELECT 
    o1.id,
    'did:firefly:org/' || o1.name,
    o2.id,
    o1.message_id,
    'ff_system',
    o1.name,
    o1.description,
    o1.profile,
    o1.created,
    o1.created
  FROM orgs as o1
  LEFT JOIN orgs o2 ON o2.identity = o1.parent;

INSERT INTO identities (
    id,
    did,
    parent,
    messages_claim,
    namespace,
    name,
    description,
    profile,
    created,
    updated
  ) SELECT 
    n.id,
    'did:firefly:node/' || n.name,
    o.id,
    n.message_id,
    'ff_system',
    n.name,
    n.description,
    n.dx_endpoint,
    n.created,
    n.created
  FROM nodes as n
  LEFT JOIN orgs o ON o.identity = n.owner;

INSERT INTO verifiers (
    id,
    namespace,
    identity,
    vtype,
    value,
    created
  ) SELECT 
    o.id,
    'ff_system',
    o.id,
    'EcdsaSecp256k1VerificationKey2019',
    o.identity,
    o.created    
  FROM orgs as o WHERE o.identity LIKE '0x%';

INSERT INTO verifiers (
    id,
    namespace,
    identity,
    vtype,
    value,
    created
  ) SELECT 
    o.id,
    'ff_system',
    o.id,
    'HyperledgerFabricMSPIdentity',
    o.identity,
    o.created
  FROM orgs as o WHERE o.identity NOT LIKE '0x%';

INSERT INTO verifiers (
    id,
    namespace,
    identity,
    vtype,
    value,
    created
  ) SELECT 
    n.id,
    'ff_system',
    n.id,
    'FireFlyDataExchangePeerId',
    n.dx_peer,
    n.created
  FROM nodes as n;

DROP TABLE orgs;
DROP TABLE nodes;

CREATE TABLE identities (
  seq            INTEGER         PRIMARY KEY AUTOINCREMENT,
  id             UUID            NOT NULL,
  did            VARCHAR(256)    NOT NULL,
  parent         UUID,
  message_id     UUID            NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  name           VARCHAR(64)     NOT NULL,
  description    VARCHAR(4096)   NOT NULL,
  profile        TEXT,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX identities_id ON identities(id);
CREATE UNIQUE INDEX identities_did ON identities(did);
CREATE UNIQUE INDEX identities_name ON identities(namespace, name);

INSERT INTO identities (
    id,
    did,
    parent,
    message_id,
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
    namespace,
    name,
    description,
    profile,
    created
  ) SELECT 
    n.id,
    'did:firefly:node/' || n.name,
    n.id,
    n.message_id,
    'ff_system',
    n.name,
    n.description,
    n.profile,
    n.created    
  FROM nodes as n
  LEFT JOIN orgs o ON o.identity = n.owner;

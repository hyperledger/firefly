BEGIN;

CREATE TABLE verifiers (
  seq            SERIAL          PRIMARY KEY,
  id             UUID            NOT NULL,
  identity       UUID            NOT NULL,
  vtype          VARCHAR(256)    NOT NULL,
  value          TEXT            NOT NULL,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX verifiers_id ON verifiers(id);
CREATE UNIQUE INDEX verifiers_value ON verifiers(vtype, value);
CREATE UNIQUE INDEX verifiers_identity ON verifiers(identity);

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
    n.dx_peer_id,
    n.created
  FROM nodes as n;

COMMIT;
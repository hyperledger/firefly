BEGIN;
CREATE TABLE nodes (
  seq            SERIAL          PRIMARY KEY,
  id             UUID            NOT NULL,  
  owner          UUID            NOT NULL,
  identity       VARCHAR(1024)   NOT NULL,
  name           VARCHAR(64)     NOT NULL,
  description    VARCHAR(4096)   NOT NULL,
  endpoint       BYTEA,
  created        BIGINT          NOT NULL,
  confirmed      BIGINT
);

CREATE UNIQUE INDEX nodes_id ON nodes(id);
CREATE UNIQUE INDEX nodes_name ON nodes(name);
CREATE UNIQUE INDEX nodes_owner ON nodes(owner);

COMMIT;
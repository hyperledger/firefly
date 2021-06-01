BEGIN;
CREATE TABLE nodes (
  seq            SERIAL          PRIMARY KEY,
  id             UUID            NOT NULL,  
  message_id     UUID            NOT NULL,
  owner          VARCHAR(1024)   NOT NULL,
  identity       VARCHAR(1024)   NOT NULL,
  description    VARCHAR(4096)   NOT NULL,
  endpoint       BYTEA,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX nodes_id ON nodes(id);
CREATE UNIQUE INDEX nodes_identity ON nodes(identity);
CREATE UNIQUE INDEX nodes_owner ON nodes(owner);

COMMIT;
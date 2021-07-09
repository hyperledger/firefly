CREATE TABLE nodes (
  seq            INTEGER         PRIMARY KEY AUTOINCREMENT,
  id             UUID            NOT NULL,  
  message_id     UUID            NOT NULL,
  owner          VARCHAR(1024)   NOT NULL,
  name           VARCHAR(64)     NOT NULL,
  description    VARCHAR(4096)   NOT NULL,
  dx_peer        VARCHAR(256),
  dx_endpoint    BYTEA,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX nodes_id ON nodes(id);
CREATE UNIQUE INDEX nodes_owner ON nodes(owner,name);
CREATE UNIQUE INDEX nodes_peer ON nodes(dx_peer);


CREATE TABLE namespaces (
  seq         INTEGER         PRIMARY KEY AUTOINCREMENT,
  id          UUID            NOT NULL,
  message_id  UUID,
  name        VARCHAR(64)     NOT NULL,
  ntype       VARCHAR(64)     NOT NULL,
  description VARCHAR(4096),
  created     BIGINT          NOT NULL
);

CREATE UNIQUE INDEX namespaces_id ON operations(id);
CREATE UNIQUE INDEX namespaces_name ON namespaces(name);


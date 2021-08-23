BEGIN;
CREATE TABLE tokenpool (
  seq            SERIAL          PRIMARY KEY,
  id             UUID            NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  name           VARCHAR(64)     NOT NULL,
  protocol_id    VARCHAR(1024)   NOT NULL,
  type           VARCHAR(64)     NOT NULL
);

CREATE UNIQUE INDEX tokenpool_id ON tokenpool(id);
CREATE UNIQUE INDEX tokenpool_name ON tokenpool(namespace,name);

COMMIT;

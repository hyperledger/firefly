BEGIN;
DROP TABLE IF EXISTS tokenpool;
CREATE TABLE tokenpool (
  seq            SERIAL          PRIMARY KEY,
  id             UUID            NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  name           VARCHAR(64)     NOT NULL,
  protocol_id    VARCHAR(1024)   NOT NULL,
  type           VARCHAR(64)     NOT NULL,
  tx_type        VARCHAR(64)     NOT NULL,
  tx_id          UUID
);

CREATE UNIQUE INDEX tokenpool_id ON tokenpool(id);
CREATE UNIQUE INDEX tokenpool_name ON tokenpool(namespace,name);
CREATE INDEX tokenpool_fortx ON tokenpool(namespace,tx_id);

COMMIT;

BEGIN;
CREATE TABLE tokenpool (
  seq            SERIAL          PRIMARY KEY,
  id             UUID            NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  name           VARCHAR(64)     NOT NULL,
  pool_id        VARCHAR(80)     NOT NULL,
  is_fungible    SMALLINT        DEFAULT 1
);

CREATE UNIQUE INDEX tokenpool_id ON tokenpool(id);
CREATE UNIQUE INDEX tokenpool_name ON tokenpool(namespace,name);

COMMIT;

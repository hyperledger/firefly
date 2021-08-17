CREATE TABLE tokenpool (
  seq            INTEGER         PRIMARY KEY AUTOINCREMENT,
  id             UUID            NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  name           VARCHAR(64)     NOT NULL,
  pool_id        VARCHAR(80)     NOT NULL,
  type           VARCHAR(64)     NOT NULL
);


CREATE UNIQUE INDEX tokenpool_id ON tokenpool(id);
CREATE UNIQUE INDEX tokenpool_name ON tokenpool(namespace,name);

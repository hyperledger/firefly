CREATE TABLE tokenpool (
  seq            INTEGER         PRIMARY KEY AUTOINCREMENT,
  pool_id        VARCHAR(80)     NOT NULL,
  base_uri       VARCHAR(256)    NOT NULL,
  is_fungible    SMALLINT        DEFAULT 1
);

CREATE UNIQUE INDEX tokenpool_id ON tokenpool(pool_id);
CREATE UNIQUE INDEX tokenpool_uri ON tokenpool(base_uri);

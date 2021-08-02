CREATE TABLE tokenpool (
  seq            INTEGER         PRIMARY KEY AUTOINCREMENT,
  pool_id        VARCHAR(80)     NOT NULL,
  base_uri       VARCHAR(256)    NOT NULL,
  is_fungible    SMALLINT        DEFAULT 1
);

CREATE UNIQUE INDEX tokenpool_id ON tokenpool(pool_id);

CREATE TABLE tokenaccount (
  seq            INTEGER         DEFAULT 0,
  member         VARCHAR(1024)   NOT NULL,
  pool_id        VARCHAR(80)     NOT NULL,
  balance        INTEGER         DEFAULT 0,
  PRIMARY KEY(member, pool_id)
);

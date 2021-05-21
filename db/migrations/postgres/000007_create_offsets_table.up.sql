BEGIN;
CREATE SEQUENCE offsets_seq;
CREATE TABLE offsets (
  id          UUID            PRIMARY KEY NOT NULL,
  otype       VARCHAR(64)     NOT NULL,
  namespace   VARCHAR(64)     NOT NULL,
  name        VARCHAR(64)     NOT NULL,
  current     BIGINT          NOT NULL
);
CREATE UNIQUE INDEX offset_unique ON offsets(otype,namespace,name);
COMMIT;
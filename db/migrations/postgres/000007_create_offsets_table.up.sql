BEGIN;
CREATE TABLE offsets (
  seq         SERIAL          PRIMARY KEY,
  id          UUID            NOT NULL,
  otype       VARCHAR(64)     NOT NULL,
  namespace   VARCHAR(64)     NOT NULL,
  name        VARCHAR(64)     NOT NULL,
  current     BIGINT          NOT NULL
);
CREATE UNIQUE INDEX offsets_id ON offsets(id);
CREATE UNIQUE INDEX offsets_unique ON offsets(otype,namespace,name);
COMMIT;
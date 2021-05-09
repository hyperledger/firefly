BEGIN;
CREATE SEQUENCE offsets_seq;
CREATE TABLE offsets (
  seq         BIGINT          NOT NULL DEFAULT nextval('offsets_seq'),
  otype       VARCHAR(64)     NOT NULL,
  namespace   VARCHAR(64)     NOT NULL,
  name        VARCHAR(64)     NOT NULL,
  current     BIGINT          NOT NULL,
  PRIMARY KEY (otype,namespace,name)
);
COMMIT;
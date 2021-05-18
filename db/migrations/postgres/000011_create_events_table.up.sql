BEGIN;
CREATE SEQUENCE events_seq;
CREATE TABLE events (
  id             CHAR(36)        NOT NULL PRIMARY KEY,
  seq            BIGINT          NOT NULL DEFAULT nextval('events_seq'),
  etype          VARCHAR(64)     NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  ref            CHAR(36)        NOT NULL,
  created        BIGINT          NOT NULL
);

CREATE INDEX events_seek ON events(namespace,seq,etype);

COMMIT;
BEGIN;
CREATE SEQUENCE events_seq;
CREATE TABLE events (
  id             CHAR(36)        NOT NULL PRIMARY KEY,
  seq            BIGINT          NOT NULL DEFAULT nextval('events_seq'),
  etype          CHAR(64)        NOT NULL,
  ref            CHAR(36)        NOT NULL
);

CREATE INDEX events_seek ON events(etype,seq);

COMMIT;
BEGIN;
CREATE SEQUENCE events_seq;
CREATE TABLE events (
  id             UUID            NOT NULL PRIMARY KEY,
  seq            BIGINT          NOT NULL DEFAULT nextval('events_seq'),
  etype          VARCHAR(64)     NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  ref            UUID            NOT NULL,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX events_sequence ON events(seq);
CREATE UNIQUE INDEX events_created ON events(created);
CREATE INDEX events_seek ON events(namespace,seq,etype);

COMMIT;
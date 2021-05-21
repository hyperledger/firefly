BEGIN;
CREATE SEQUENCE events_seq;
CREATE TABLE events (
  seq            SERIAL          PRIMARY KEY,
  id             UUID            NOT NULL,
  etype          VARCHAR(64)     NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  ref            UUID            NOT NULL,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX events_id ON events(id);
CREATE UNIQUE INDEX events_created ON events(created);
CREATE INDEX events_seek ON events(namespace,seq,etype);

COMMIT;
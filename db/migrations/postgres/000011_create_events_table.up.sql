BEGIN;
CREATE TABLE events (
  seq            SERIAL          PRIMARY KEY,
  id             UUID            NOT NULL,
  etype          VARCHAR(64)     NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  ref            UUID,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX events_id ON events(id);
CREATE UNIQUE INDEX events_created ON events(created);

COMMIT;
CREATE TABLE events (
  id             string       NOT NULL,
  etype          string       NOT NULL,
  namespace      string       NOT NULL,
  ref            string       NOT NULL
);

CREATE UNIQUE INDEX events_primary ON events(id);
CREATE INDEX events_seek ON events(namespace,etype);

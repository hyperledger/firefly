CREATE TABLE schemas (
  id          string   NOT NULL,
  stype       string   NOT NULL,
  namespace   string   NOT NULL,
  entity      string   NOT NULL,
  version     string   NOT NULL,
  hash        string   NOT NULL,
  created     int64    NOT NULL,
  value       blob
);

CREATE UNIQUE INDEX schemas_primary ON schemas(id);
CREATE UNIQUE INDEX schemas_unique ON schemas(namespace,entity,version);
CREATE INDEX schemas_search ON schemas(namespace,entity,version,created);
CREATE INDEX schemas_hash ON schemas(namespace,hash);

CREATE TABLE datadefs (
  id          string   NOT NULL,
  validator   string   NOT NULL,
  namespace   string   NOT NULL,
  name        string   NOT NULL,
  version     string   NOT NULL,
  hash        string   NOT NULL,
  created     int64    NOT NULL,
  value       blob
);

CREATE UNIQUE INDEX datadefs_primary ON datadefs(id);
CREATE UNIQUE INDEX datadefs_unique ON datadefs(namespace,name,version);
CREATE INDEX datadefs_search ON datadefs(namespace,name,version,created);
CREATE INDEX datadefs_hash ON datadefs(namespace,hash);

CREATE TABLE datatypes (
  id          string   NOT NULL,
  validator   string   NOT NULL,
  namespace   string   NOT NULL,
  name        string   NOT NULL,
  version     string   NOT NULL,
  hash        string   NOT NULL,
  created     int64    NOT NULL,
  value       blob
);

CREATE UNIQUE INDEX datatypes_primary ON datatypes(id);
CREATE UNIQUE INDEX datatypes_unique ON datatypes(namespace,name,version);
CREATE INDEX datatypes_created ON datatypes(created);

CREATE TABLE data (
  id          string    NOT NULL,
  dtype       string    NOT NULL,
  namespace   string    NOT NULL,
  entity      string    NOT NULL,
  schema      string    NOT NULL,
  hash        string    NOT NULL,
  created     int64     NOT NULL,
  value       BLOB      NOT NULL
);

CREATE UNIQUE INDEX data_primary ON data(id);
CREATE INDEX data_search ON data(namespace,dtype,entity,schema,hash,created);
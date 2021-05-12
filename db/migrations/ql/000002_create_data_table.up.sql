CREATE TABLE data (
  id             string    NOT NULL,
  validator      string    NOT NULL,
  namespace      string    NOT NULL,
  def_name       string    NOT NULL,
  def_version    string    NOT NULL,
  hash           string    NOT NULL,
  created        int64     NOT NULL,
  value          BLOB      NOT NULL
);

CREATE UNIQUE INDEX data_primary ON data(id);
CREATE INDEX data_search ON data(namespace,validator,def_name,def_version,hash,created);
CREATE TABLE batches (
  id          string   NOT NULL,
  btype       string   NOT NULL,
  namespace   string   NOT NULL,
  author      string   NOT NULL,
  hash        string   NOT NULL,
  created     int64    NOT NULL,
  payload     blob     NOT NULL,
  payload_ref string,
  confirmed   int64    NOT NULL,
  tx_type     string   NOT NULL,
  tx_id       string,
);

CREATE UNIQUE INDEX batches_primary ON batches(id);
CREATE INDEX batches_search ON batches(namespace,btype,author,confirmed,created);
CREATE INDEX batches_fortx ON batches(namespace,tx_id);

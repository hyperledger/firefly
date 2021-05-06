CREATE TABLE transactions (
  id          string   NOT NULL,
  ttype       string   NOT NULL,
  namespace   string   NOT NULL,
  author      string   NOT NULL,
  created     int64    NOT NULL,
  tracking_id string,
  protocol_id string,
  confirmed   int64    NOT NULL,
  info        blob
);

CREATE UNIQUE INDEX transactions_primary ON transactions(id);
CREATE INDEX transactions_search ON transactions(namespace,ttype,author,confirmed,created);
CREATE INDEX transactions_tracking_id ON transactions(tracking_id);
CREATE INDEX transactions_protocol_id ON transactions(protocol_id);


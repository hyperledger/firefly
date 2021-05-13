CREATE TABLE transactions (
  id          string   NOT NULL,
  ttype       string   NOT NULL,
  namespace   string   NOT NULL,
  msg_id      string,
  batch_id    string,
  author      string   NOT NULL,
  hash        string   NOT NULL,
  created     int64    NOT NULL,
  protocol_id string,
  status      string   NOT NULL,
  confirmed   int64    NOT NULL,
  info        blob
);

CREATE UNIQUE INDEX transactions_primary ON transactions(id);
CREATE INDEX transactions_search ON transactions(namespace,ttype,author,status,confirmed,created);
CREATE INDEX transactions_protocol_id ON transactions(protocol_id);


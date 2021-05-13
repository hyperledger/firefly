CREATE TABLE messages (
  id          string  NOT NULL,
  cid         string,
  mtype       string  NOT NULL,
  author      string  NOT NULL,
  created     int64   NOT NULL,
  namespace   string  NOT NULL,
  topic       string  NOT NULL,
  context     string  NOT NULL,
  group_id    string,
  datahash    string  NOT NULL,
  hash        string  NOT NULL,
  confirmed   int64   NOT NULL,
  tx_type     string  NOT NULL,
  tx_id       string,
  batch_id    string
);

CREATE UNIQUE INDEX messages_primary ON messages(id);
CREATE INDEX messages_search ON messages(namespace,mtype,confirmed,context,topic,group_id,author,cid,hash,created );
CREATE INDEX messages_batch ON messages(namespace,batch_id);
CREATE INDEX messages_tx ON messages(namespace,tx_type,tx_id);
CREATE TABLE messages (
  id          string  NOT NULL,
  cid         string,
  type        string  NOT NULL,
  author      string  NOT NULL,
  created     int64   NOT NULL,
  namespace   string  NOT NULL,
  topic       string  NOT NULL,
  context     string  NOT NULL,
  group_id    string,
  datahash    string  NOT NULL,
  hash        string  NOT NULL,
  confirmed   int64,
  tx_id       string
);

CREATE UNIQUE INDEX messages_primary ON messages(id);
CREATE INDEX messages_search ON messages(namespace,type,confirmed,context,topic,group_id,author,cid,hash,created );
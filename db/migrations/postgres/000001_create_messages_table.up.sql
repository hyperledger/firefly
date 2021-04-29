CREATE TABLE messages (
  id          CHAR(36)        NOT NULL PRIMARY KEY,
  cid         CHAR(36),
  type        VARCHAR(64)     NOT NULL,
  author      VARCHAR(1024)   NOT NULL,
  created     BIGINT          NOT NULL,
  namespace   VARCHAR(64)     NOT NULL,
  topic       VARCHAR(128)    NOT NULL,
  context     VARCHAR(1024)   NOT NULL,
  group_id    CHAR(36),
  datahash    CHAR(32)        NOT NULL,
  hash        CHAR(32)        NOT NULL,
  confirmed   BIGINT,
  tx_id       CHAR(36)
);

CREATE INDEX messages_search ON messages(namespace,type,confirmed,context,topic,group_id,author,cid,hash,created );
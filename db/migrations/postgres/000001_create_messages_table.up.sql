BEGIN;
CREATE SEQUENCE messages_seq;
CREATE TABLE messages (
  id          CHAR(36)        NOT NULL PRIMARY KEY,
  seq         BIGINT          NOT NULL DEFAULT nextval('messages_seq'),
  cid         CHAR(36),
  mtype       VARCHAR(64)     NOT NULL,
  author      VARCHAR(1024)   NOT NULL,
  created     BIGINT          NOT NULL,
  namespace   VARCHAR(64)     NOT NULL,
  topic       VARCHAR(128)    NOT NULL,
  context     VARCHAR(1024)   NOT NULL,
  group_id    CHAR(36),
  datahash    CHAR(64)        NOT NULL,
  hash        CHAR(64)        NOT NULL,
  confirmed   BIGINT          NOT NULL,
  tx_type     VARCHAR(64)     NOT NULL,
  tx_id       CHAR(36),
  batch_id    CHAR(36)
);

CREATE INDEX messages_search ON messages(namespace,mtype,confirmed,context,topic,group_id,author,cid,hash,created);
CREATE INDEX messages_batch ON messages(namespace,batch_id);
CREATE INDEX messages_tx ON messages(namespace,tx_type,tx_id);
COMMIT;
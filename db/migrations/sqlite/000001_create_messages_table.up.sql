CREATE TABLE messages (
  seq         INTEGER         PRIMARY KEY AUTOINCREMENT,
  id          UUID            NOT NULL,
  cid         CHAR(36),
  mtype       VARCHAR(64)     NOT NULL,
  author      VARCHAR(1024)   NOT NULL,
  created     BIGINT          NOT NULL,
  namespace   VARCHAR(64)     NOT NULL,
  topics      VARCHAR(1024)   NOT NULL,
  tag         VARCHAR(64)     NOT NULL,
  group_hash  CHAR(64),
  datahash    CHAR(64)        NOT NULL,
  hash        CHAR(64)        NOT NULL,
  pins        VARCHAR(1024)   NOT NULL,
  confirmed   BIGINT,
  tx_type     VARCHAR(64)     NOT NULL,
  batch_id    UUID,
  local       BOOLEAN         NOT NULL,
  pending     SMALLINT        NOT NULL,
  rejected    BOOLEAN         NOT NULL
);

CREATE UNIQUE INDEX messages_id ON messages(id);
CREATE INDEX messages_sortorder ON messages(pending, confirmed, created);
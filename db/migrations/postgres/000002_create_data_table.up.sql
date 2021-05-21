BEGIN;
CREATE SEQUENCE data_seq;
CREATE TABLE data (
  id             UUID            NOT NULL PRIMARY KEY,
  seq            BIGINT          NOT NULL DEFAULT nextval('data_seq'),
  validator      VARCHAR(64)     NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  def_name       VARCHAR(64)     NOT NULL,
  def_version    VARCHAR(64)     NOT NULL,
  hash           CHAR(64)        NOT NULL,
  created        BIGINT          NOT NULL,
  value          JSONB           NOT NULL
);
CREATE UNIQUE INDEX data_sequence ON data(seq);
CREATE INDEX data_hash ON data(namespace,hash);
CREATE INDEX data_created ON data(namespace,created);
COMMIT;
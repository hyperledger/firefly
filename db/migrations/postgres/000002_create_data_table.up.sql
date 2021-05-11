BEGIN;
CREATE SEQUENCE data_seq;
CREATE TABLE data (
  id             CHAR(36)        NOT NULL PRIMARY KEY,
  seq            BIGINT          NOT NULL DEFAULT nextval('data_seq'),
  validator      VARCHAR(64)     NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  def_name       VARCHAR(64)     NOT NULL,
  def_version    VARCHAR(64)     NOT NULL,
  hash           CHAR(64)        NOT NULL,
  created        BIGINT          NOT NULL,
  value          JSONB           NOT NULL
);
CREATE INDEX data_search ON data(namespace,validator,def_name,def_version,hash,created);
COMMIT;
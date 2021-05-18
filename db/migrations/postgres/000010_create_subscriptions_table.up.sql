BEGIN;
CREATE SEQUENCE subscriptions_seq;
CREATE TABLE subscriptions (
  id             CHAR(36)        NOT NULL PRIMARY KEY,
  seq            BIGINT          NOT NULL DEFAULT nextval('subscriptions_seq'),
  namespace      VARCHAR(64)     NOT NULL,
  name           VARCHAR(64)     NOT NULL,
  dispatcher     VARCHAR(64)     NOT NULL,
  events         VARCHAR(256)    NOT NULL,
  filter_topic   VARCHAR(256)    NOT NULL,
  filter_context VARCHAR(256)    NOT NULL,
  filter_group   VARCHAR(256)    NOT NULL,
  options        JSONB           NOT NULL,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX subscriptions_name ON subscriptions(namespace,name);

COMMIT;
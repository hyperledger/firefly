BEGIN;
CREATE TABLE subscriptions (
  seq            SERIAL          PRIMARY KEY,
  id             UUID            NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  name           VARCHAR(64)     NOT NULL,
  transport      VARCHAR(64)     NOT NULL,
  filter_events  VARCHAR(256)    NOT NULL,
  filter_topic   VARCHAR(256)    NOT NULL,
  filter_context VARCHAR(256)    NOT NULL,
  filter_group   VARCHAR(256)    NOT NULL,
  options        JSONB           NOT NULL,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX subscriptions_id ON subscriptions(id);
CREATE UNIQUE INDEX subscriptions_name ON subscriptions(namespace,name);

COMMIT;
CREATE TABLE contractevents (
  seq              INTEGER         PRIMARY KEY AUTOINCREMENT,
  id               UUID            NOT NULL,
  namespace        VARCHAR(64)     NOT NULL,
  name             VARCHAR(1024)   NOT NULL,
  subscription_id  UUID            NOT NULL,
  outputs          TEXT,
  info             TEXT,
  timestamp        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX contractevents_name ON contractevents(namespace,name);
CREATE UNIQUE INDEX contractevents_timestamp ON contractevents(timestamp);
CREATE UNIQUE INDEX contractevents_subscription_id ON contractevents(subscription_id);
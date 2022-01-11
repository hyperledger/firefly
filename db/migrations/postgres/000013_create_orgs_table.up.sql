BEGIN;
CREATE TABLE orgs (
  seq            SERIAL          PRIMARY KEY,
  id             UUID            NOT NULL,
  message_id     UUID            NOT NULL,
  name           VARCHAR(64)     NOT NULL,
  parent         VARCHAR(1024),
  identity       VARCHAR(1024)   NOT NULL,
  description    VARCHAR(4096)   NOT NULL,
  profile        TEXT,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX orgs_id ON orgs(id);
CREATE UNIQUE INDEX orgs_identity ON orgs(identity);
CREATE UNIQUE INDEX orgs_name ON orgs(name);

COMMIT;
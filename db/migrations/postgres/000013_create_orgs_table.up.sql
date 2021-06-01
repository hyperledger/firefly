BEGIN;
CREATE TABLE orgs (
  seq            SERIAL          PRIMARY KEY,
  id             UUID            NOT NULL,
  parent         UUID,
  identity       VARCHAR(1024)   NOT NULL,
  description    VARCHAR(4096)   NOT NULL,
  profile        BYTEA,
  created        BIGINT          NOT NULL,
  confirmed      BIGINT
);

CREATE UNIQUE INDEX orgs_id ON orgs(id);
CREATE UNIQUE INDEX orgs_identity ON orgs(identity);

COMMIT;
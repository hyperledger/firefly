BEGIN;
CREATE TABLE members (
  seq            SERIAL          PRIMARY KEY,
  group_hash     CHAR(64)        NOT NULL,
  idx            INT             NOT NULL,
  identity       VARCHAR(1024)   NOT NULL,
  node_id        UUID            NOT NULL
);

CREATE INDEX members_group ON members(group_hash);

COMMIT;
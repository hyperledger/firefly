BEGIN;
CREATE TABLE members (
  seq            SERIAL          PRIMARY KEY,
  group_id       UUID            NOT NULL REFERENCES groups(id),
  idx            INT             NOT NULL,
  identity       VARCHAR(1024)   NOT NULL,
  node_id        UUID            NOT NULL
);

CREATE INDEX members_group ON members(group_id);

COMMIT;
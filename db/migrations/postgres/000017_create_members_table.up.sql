BEGIN;
CREATE TABLE members (
  seq            SERIAL          PRIMARY KEY,
  group_id       UUID            NOT NULL REFERENCES groups(id),
  idx            INT             NOT NULL,
  org            UUID            NOT NULL,
  node           UUID            NOT NULL
);

CREATE INDEX members_group ON members(group_id);

COMMIT;
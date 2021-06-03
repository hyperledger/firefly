BEGIN;
CREATE TABLE recipients (
  seq            SERIAL          PRIMARY KEY,
  group_id       UUID            NOT NULL REFERENCES groups(id),
  idx            INT             NOT NULL,
  org            UUID            NOT NULL,
  node           UUID            NOT NULL
);

CREATE INDEX recipients_group ON recipients(group_id);

COMMIT;
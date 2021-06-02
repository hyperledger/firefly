BEGIN;
CREATE TABLE recipients (
  seq            SERIAL          PRIMARY KEY,
  group_id       UUID            NOT NULL REFERENCES groups(id),
  idx            INT             NOT NULL,
  identity       VARCHAR(1024)   NOT NULL
);

CREATE UNIQUE INDEX recipients_group ON recipients(group_id);

COMMIT;
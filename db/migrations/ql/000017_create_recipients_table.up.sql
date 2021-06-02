CREATE TABLE recipients (
  group_id       string          NOT NULL,
  idx            int64           NOT NULL,
  identity       string          NOT NULL
);

CREATE UNIQUE INDEX recipients_id ON recipients(id);

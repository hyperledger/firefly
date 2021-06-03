CREATE TABLE recipients (
  group_id       string          NOT NULL,
  idx            int64           NOT NULL,
  org            string          NOT NULL,
  node           string          NOT NULL
);

CREATE INDEX recipients_group ON recipients(group_id);

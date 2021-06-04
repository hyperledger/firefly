CREATE TABLE members (
  group_id       string          NOT NULL,
  idx            int64           NOT NULL,
  org            string          NOT NULL,
  node           string          NOT NULL
);

CREATE INDEX members_group ON members(group_id);

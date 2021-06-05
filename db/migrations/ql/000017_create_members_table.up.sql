CREATE TABLE members (
  group_id       string          NOT NULL,
  idx            int64           NOT NULL,
  identity       string          NOT NULL,
  node_id        string          NOT NULL
);

CREATE INDEX members_group ON members(group_id);

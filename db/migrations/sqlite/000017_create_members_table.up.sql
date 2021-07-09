CREATE TABLE members (
  seq            INTEGER         PRIMARY KEY AUTOINCREMENT,
  group_hash     CHAR(64)        NOT NULL,
  idx            INT             NOT NULL,
  identity       VARCHAR(1024)   NOT NULL,
  node_id        UUID            NOT NULL
);

CREATE INDEX members_group ON members(group_hash);


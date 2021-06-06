CREATE TABLE groupcontexts (
  hash           string          NOT NULL,
  nonce          int64           NOT NULL,
  group_id       string          NOT NULL,
  topic          string          NOT NULL
);

CREATE INDEX groupcontexts_hash ON groupcontexts(hash);
CREATE INDEX groupcontexts_group ON groupcontexts(group_id);

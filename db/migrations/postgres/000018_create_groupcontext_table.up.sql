BEGIN;
CREATE TABLE groupcontexts (
  seq            SERIAL          PRIMARY KEY,
  hash           CHAR(64)        NOT NULL,
  nonce          BIGINT          NOT NULL,
  group_id       UUID            NOT NULL,
  topic          VARCHAR(64)     NOT NULL
);

CREATE INDEX groupcontexts_hash ON groupcontexts(hash);
CREATE INDEX groupcontexts_group ON groupcontexts(group_id);

COMMIT;
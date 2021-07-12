CREATE TABLE groups (
  seq            INTEGER         PRIMARY KEY AUTOINCREMENT,
  message_id     UUID,
  name           VARCHAR(64)     NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  ledger         UUID,
  hash           CHAR(64)        NOT NULL,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX groups_hash ON groups(hash);


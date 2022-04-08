BEGIN;
CREATE TABLE groups (
  seq            SERIAL          PRIMARY KEY,
  message_id     UUID,
  name           VARCHAR(64)     NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  hash           CHAR(64)        NOT NULL,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX groups_hash ON groups(hash);

COMMIT;
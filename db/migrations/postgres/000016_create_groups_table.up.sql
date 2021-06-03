BEGIN;
CREATE TABLE groups (
  seq            SERIAL          PRIMARY KEY,
  id             UUID            NOT NULL,
  message_id     UUID,
  namespace      VARCHAR(64)     NOT NULL,
  description    VARCHAR(4096)   NOT NULL,
  ledger         UUID,
  hash           CHAR(64)        NOT NULL,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX groups_id ON groups(id);
CREATE UNIQUE INDEX groups_hash ON groups(namespace,ledger,hash);

COMMIT;
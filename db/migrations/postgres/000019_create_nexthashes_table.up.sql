BEGIN;
CREATE TABLE nexthashes (
  seq            SERIAL          PRIMARY KEY,
  context        CHAR(64)        NOT NULL,
  identity       VARCHAR(1024)   NOT NULL,
  hash           CHAR(64)        NOT NULL,
  nonce          BIGINT          NOT NULL
);

CREATE INDEX nexthashes_hash ON nexthashes(hash);

COMMIT;
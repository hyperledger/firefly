BEGIN;
CREATE TABLE contexts (
  seq            SERIAL          PRIMARY KEY,
  hash           CHAR(64)        NOT NULL,
  nonce          BIGINT          NOT NULL,
  group_id       UUID            NOT NULL,
  topic          VARCHAR(64)     NOT NULL
);

CREATE INDEX contexts_hash ON contexts(hash);

COMMIT;
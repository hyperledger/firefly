BEGIN;
CREATE TABLE tokenaccount (
  seq            SERIAL          PRIMARY KEY,
  protocol_id    VARCHAR(1024)   NOT NULL,
  token_index    VARCHAR(1024)   NOT NULL,
  identity       VARCHAR(1024)   NOT NULL,
  balance        BIGINT          DEFAULT 0
);

CREATE INDEX tokenaccount_pool ON tokenaccount(protocol_id,token_index,identity);

COMMIT;

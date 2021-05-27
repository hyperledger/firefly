BEGIN;
CREATE TABLE blocked (
  seq            SERIAL          PRIMARY KEY,
  id             UUID            NOT NULL,  
  namespace      VARCHAR(64)     NOT NULL,
  context        VARCHAR(1024)   NOT NULL,
  group_id       UUID,
  message_id     UUID            NOT NULL,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX blocked_id ON blocked(id);
CREATE UNIQUE INDEX blocked_message ON blocked(message_id);

COMMIT;
CREATE TABLE groups (
  message_id     string,
  name           string          NOT NULL,
  namespace      string          NOT NULL,
  ledger         string,
  hash           string          NOT NULL,
  created        int64           NOT NULL
);

CREATE UNIQUE INDEX groups_hash ON groups(namespace,ledger,hash);

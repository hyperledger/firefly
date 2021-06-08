CREATE TABLE groups (
  id             string          NOT NULL,
  message_id     string,
  namespace      string          NOT NULL,
  description    string          NOT NULL,
  ledger         string,
  hash           string          NOT NULL,
  created        int64           NOT NULL
);

CREATE UNIQUE INDEX groups_id ON groups(id);
CREATE UNIQUE INDEX groups_hash ON groups(namespace,ledger,hash);

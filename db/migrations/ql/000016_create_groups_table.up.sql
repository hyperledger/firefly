CREATE TABLE groups (
  id             string          NOT NULL,
  message_id     string,
  namespace      string          NOT NULL,
  description    string          NOT NULL,
  ledger         string,
  created        int64           NOT NULL
);

CREATE UNIQUE INDEX groups_id ON groups(id);

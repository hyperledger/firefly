CREATE TABLE namespaces (
  id          string          NOT NULL,
  message_id  string          NOT NULL,
  name        string          NOT NULL,
  ntype       string          NOT NULL,
  description string,
  created     int64           NOT NULL
);

CREATE UNIQUE INDEX namespaces_primary ON namespaces(id);
CREATE UNIQUE INDEX namespaces_name ON namespaces(name);

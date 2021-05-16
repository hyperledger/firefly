CREATE TABLE namespaces (
  id          string          NOT NULL,
  name        string          NOT NULL,
  ntype       string          NOT NULL,
  description string,
  created     int64           NOT NULL,
  confirmed   int64
);

CREATE UNIQUE INDEX namespaces_primary ON namespaces(id);
CREATE UNIQUE INDEX namespaces_name ON namespaces(name);

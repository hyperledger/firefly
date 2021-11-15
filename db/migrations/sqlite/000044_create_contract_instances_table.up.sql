CREATE TABLE contract_interfaces (
  seq               INTEGER         PRIMARY KEY AUTOINCREMENT,
  id                UUID            NOT NULL,
  namespace         VARCHAR(64)     NOT NULL,
  name              VARCHAR(1024)   NOT NULL,
  version           VARCHAR(64)     NOT NULL,
);

CREATE UNIQUE INDEX contract_interfaces_id ON contract_interfaces(id);
CREATE UNIQUE INDEX contract_interfaces_namespace_name_version ON contract_interfaces(namespace,name,version);

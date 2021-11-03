CREATE TABLE contract_instances (
  seq               INTEGER         PRIMARY KEY AUTOINCREMENT,
  id                UUID            NOT NULL,
  contract_id       UUID            NOT NULL,
  namespace         VARCHAR(64)     NOT NULL,
  name              VARCHAR(1024),
  onchain_location  VARCHAR(2048)   NOT NULL,
  FOREIGN KEY(contract_id) REFERENCES contracts(id)
);

CREATE UNIQUE INDEX contract_instances_id ON contract_instances(id);
CREATE UNIQUE INDEX contract_instances_namespace_name ON contract_instances(namespace,name);

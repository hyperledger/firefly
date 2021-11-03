CREATE TABLE contract_definitions (
  seq               INTEGER         PRIMARY KEY AUTOINCREMENT,
  id                UUID            NOT NULL,
  namespace         VARCHAR(64)     NOT NULL,
  name              VARCHAR(1024)   NOT NULL,
  version           VARCHAR(64)     NOT NULL,
  ffabi             BYTEA,
  onchain_location  VARCHAR(2048)
);

CREATE UNIQUE INDEX contract_definitions_id ON contract_definitions(id);
CREATE UNIQUE INDEX contract_definitions_namespace_name_version ON contract_definitions(namespace,name,version);

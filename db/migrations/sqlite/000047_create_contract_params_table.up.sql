CREATE TABLE contract_params (
  seq               INTEGER         PRIMARY KEY AUTOINCREMENT,
  interface_id      UUID            NOT NULL,
  parent_name       VARCHAR(64)     NOT NULL,
  namespace         VARCHAR(64)     NOT NULL,
  name              VARCHAR(64)     NOT NULL,
  type              VARCHAR(64)     NOT NULL,
  index             INTEGER         NOT NULL,
  role              VARCHAR(64)     NOT NULL,
);

CREATE UNIQUE INDEX contract_interfaces_id ON contract_definitions(id);
CREATE UNIQUE INDEX contract_interfaces_interface_id_parent_id_name ON contract_definitions(interface_id,parent_id,name);

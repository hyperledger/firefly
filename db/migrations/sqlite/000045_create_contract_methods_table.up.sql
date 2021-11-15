CREATE TABLE contract_methods (
  seq               INTEGER         PRIMARY KEY AUTOINCREMENT,
  interface_id      UUID            NOT NULL,
  namespace         VARCHAR(64)     NOT NULL,
  name              VARCHAR(64)     NOT NULL,
);

CREATE UNIQUE INDEX contract_methods_id ON contract_methods(id);
CREATE UNIQUE INDEX contract_methods_interface_id_name ON contract_methods(interface_id,name);

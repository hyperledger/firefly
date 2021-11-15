CREATE TABLE contractmethods (
  seq               INTEGER         PRIMARY KEY AUTOINCREMENT,
  interface_id      UUID            NOT NULL,
  namespace         VARCHAR(64)     NOT NULL,
  parent_name       VARCHAR(64)     NOT NULL,
  name              VARCHAR(64)     NOT NULL
);

CREATE UNIQUE INDEX contract_methods_interface_id_name ON contractmethods(interface_id,name);

CREATE TABLE contractinterface_methods (
  seq               INTEGER         PRIMARY KEY AUTOINCREMENT,
  id                UUID            NOT NULL,
  interface_id      UUID            NOT NULL,
  namespace         VARCHAR(64)     NOT NULL,
  name              VARCHAR(64)     NOT NULL,
  params            BYTEA           NOT NULL,
  returns           BYTEA           NOT NULL
);

CREATE UNIQUE INDEX contractmethods_interface_id_name ON contractinterface_methods(interface_id,name);

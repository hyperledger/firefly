CREATE TABLE contractparams (
  seq               INTEGER         PRIMARY KEY AUTOINCREMENT,
  interface_id      UUID            NOT NULL,
  parent_name       VARCHAR(64)     NOT NULL,
  namespace         VARCHAR(64)     NOT NULL,
  name              VARCHAR(64)     NOT NULL,
  type              VARCHAR(64)     NOT NULL,
  param_index       INTEGER         NOT NULL,
  role              VARCHAR(64)     NOT NULL
);

CREATE UNIQUE INDEX contractparams_interface_id_parent_name_name ON contractparams(interface_id,parent_name,name);

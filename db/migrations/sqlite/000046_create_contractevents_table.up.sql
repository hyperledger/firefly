CREATE TABLE contractevents (
  seq               INTEGER         PRIMARY KEY AUTOINCREMENT,
  interface_id      UUID            NOT NULL,
  namespace         VARCHAR(64)     NOT NULL,
  parent_name       VARCHAR(64)     NOT NULL,
  name              VARCHAR(64)     NOT NULL
);

CREATE UNIQUE INDEX contractevents_interface_id_name ON contractevents(interface_id,name);

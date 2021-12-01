CREATE TABLE contractinterface_events (
  seq               INTEGER         PRIMARY KEY AUTOINCREMENT,
  id                UUID            NOT NULL,
  interface_id      UUID            NULL,
  namespace         VARCHAR(64)     NOT NULL,
  name              VARCHAR(64)     NOT NULL,
  params            BYTEA           NOT NULL
);

CREATE UNIQUE INDEX contractevents_interface_id_name ON contractinterface_events(interface_id,name);

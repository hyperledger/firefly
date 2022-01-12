CREATE TABLE ffievents (
  seq               INTEGER         PRIMARY KEY AUTOINCREMENT,
  id                UUID            NOT NULL,
  interface_id      UUID            NULL,
  namespace         VARCHAR(64)     NOT NULL,
  name              VARCHAR(1024)   NOT NULL,
  pathname          VARCHAR(1024)   NOT NULL,
  description       TEXT            NOT NULL,
  params            TEXT           NOT NULL
);

CREATE UNIQUE INDEX ffievents_pathname ON ffievents(interface_id,pathname);

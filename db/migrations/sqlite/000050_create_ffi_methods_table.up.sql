CREATE TABLE ffimethods (
  seq               INTEGER         PRIMARY KEY AUTOINCREMENT,
  id                UUID            NOT NULL,
  interface_id      UUID            NOT NULL,
  namespace         VARCHAR(64)     NOT NULL,
  name              VARCHAR(1024)   NOT NULL,
  pathname          VARCHAR(1024)   NOT NULL,
  description       TEXT            NOT NULL,
  params            TEXT           NOT NULL,
  returns           TEXT           NOT NULL
);

CREATE UNIQUE INDEX ffimethods_pathname ON ffimethods(interface_id,pathname);

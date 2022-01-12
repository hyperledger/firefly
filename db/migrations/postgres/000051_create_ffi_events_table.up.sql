BEGIN;
CREATE TABLE ffievents (
  seq               SERIAL          PRIMARY KEY,
  id                UUID            NOT NULL,
  interface_id      UUID            NULL,
  namespace         VARCHAR(64)     NOT NULL,
  name              VARCHAR(1024)   NOT NULL,
  pathname          VARCHAR(1024)   NOT NULL,
  description       TEXT            NOT NULL,
  params            BYTEA           NOT NULL
);

CREATE UNIQUE INDEX ffievents_pathname ON ffievents(interface_id,pathname);
COMMIT;
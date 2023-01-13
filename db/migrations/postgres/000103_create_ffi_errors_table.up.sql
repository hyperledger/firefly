BEGIN;
CREATE TABLE ffierrors (
  seq               SERIAL          PRIMARY KEY,
  id                UUID            NOT NULL,
  interface_id      UUID            NULL,
  namespace         VARCHAR(64)     NOT NULL,
  name              VARCHAR(1024)   NOT NULL,
  pathname          VARCHAR(1024)   NOT NULL,
  description       TEXT            NOT NULL,
  params            TEXT            NOT NULL
);

CREATE UNIQUE INDEX ffierrors_pathname ON ffierrors(interface_id,pathname);
COMMIT;
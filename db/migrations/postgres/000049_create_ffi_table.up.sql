BEGIN;
CREATE TABLE ffi (
  seq               SERIAL          PRIMARY KEY,
  id                UUID            NOT NULL,
  namespace         VARCHAR(64)     NOT NULL,
  name              VARCHAR(1024)   NOT NULL,
  version           VARCHAR(64)     NOT NULL,
  description       TEXT            NOT NULL,
  message_id        UUID            NOT NULL
);

CREATE UNIQUE INDEX ffi_id ON ffi(id);
CREATE UNIQUE INDEX ffi_name ON ffi(namespace,name,version);
COMMIT;
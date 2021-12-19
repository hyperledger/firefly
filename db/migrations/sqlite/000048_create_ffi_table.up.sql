CREATE TABLE ffi (
  seq               INTEGER         PRIMARY KEY AUTOINCREMENT,
  id                UUID            NOT NULL,
  namespace         VARCHAR(64)     NOT NULL,
  name              VARCHAR(1024)   NOT NULL,
  version           VARCHAR(64)     NOT NULL,
  description       VARCHAR(65536)  NOT NULL
);

CREATE UNIQUE INDEX ffi_id ON ffi(id);
CREATE UNIQUE INDEX ffi_name ON ffi(namespace,name,version);

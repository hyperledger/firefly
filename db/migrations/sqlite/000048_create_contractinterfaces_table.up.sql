CREATE TABLE contractinterfaces (
  seq               INTEGER         PRIMARY KEY AUTOINCREMENT,
  id                UUID            NOT NULL,
  namespace         VARCHAR(64)     NOT NULL,
  name              VARCHAR(1024)   NOT NULL,
  version           VARCHAR(64)     NOT NULL
);

CREATE UNIQUE INDEX contractinterfaces_id ON contractinterfaces(id);
CREATE UNIQUE INDEX contractinterfaces_namespace_name_version ON contractinterfaces(namespace,name,version);

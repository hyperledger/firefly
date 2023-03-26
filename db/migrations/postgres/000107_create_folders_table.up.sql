BEGIN;
CREATE TABLE folders (
  seq               SERIAL          PRIMARY KEY,
  id                UUID            NOT NULL,
  name              TEXT            NOT NULL,
  path              TEXT            NOT NULL,
);

CREATE UNIQUE INDEX folders_path ON folders(path, name);
COMMIT;
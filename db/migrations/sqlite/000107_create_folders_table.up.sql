CREATE TABLE folders (
  seq               INTEGER         PRIMARY KEY AUTOINCREMENT,
  id                UUID            NOT NULL,
  name              TEXT            NOT NULL,
  path              TEXT            NOT NULL,
);

CREATE UNIQUE INDEX folders_path ON folders(path);

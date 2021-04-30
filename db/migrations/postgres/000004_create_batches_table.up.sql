CREATE TABLE batches (
  id          CHAR(36)   NOT NULL PRIMARY KEY,
  author      VARCHAR(1024)   NOT NULL,
  hash        CHAR(32)   NOT NULL,
  created     INTEGER    NOT NULL,
  payload     BLOB       NOT NULL
);

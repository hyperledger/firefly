CREATE TABLE batches (
  id          string   NOT NULL,
  author      string   NOT NULL,
  hash        string   NOT NULL,
  created     int64    NOT NULL,
  payload     blob     NOT NULL
);

CREATE UNIQUE INDEX batches_primary ON batches(id);

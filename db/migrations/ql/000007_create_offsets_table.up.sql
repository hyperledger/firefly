CREATE TABLE offsets (
  id          string   NOT NULL,
  otype       string   NOT NULL,
  namespace   string   NOT NULL,
  name        string   NOT NULL,
  current     int64    NOT NULL,
);
CREATE UNIQUE INDEX offsets_primary ON offsets(id);
CREATE UNIQUE INDEX offsets_unique ON offsets(otype,namespace,name);

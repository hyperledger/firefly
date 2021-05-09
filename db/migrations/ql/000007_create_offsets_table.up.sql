CREATE TABLE offsets (
  otype       string   NOT NULL,
  namespace   string   NOT NULL,
  name        string   NOT NULL,
  current     int64    NOT NULL,
);
CREATE UNIQUE INDEX offsets_primary ON offsets(otype,namespace,name);

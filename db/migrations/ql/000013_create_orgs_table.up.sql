CREATE TABLE orgs (
  id             string            NOT NULL,  
  parent         string,
  identity       string            NOT NULL,
  name           string            NOT NULL,
  description    string            NOT NULL,
  profile        blob,
  created        int64             NOT NULL,
  confirmed      int64
);

CREATE UNIQUE INDEX orgs_id ON orgs(id);
CREATE UNIQUE INDEX orgs_name ON orgs(name);

CREATE TABLE orgs (
  id             string            NOT NULL,  
  message_id     string            NOT NULL,
  parent         string,
  identity       string            NOT NULL,
  description    string            NOT NULL,
  profile        blob,
  created        int64             NOT NULL
);

CREATE UNIQUE INDEX orgs_id ON orgs(id);
CREATE UNIQUE INDEX orgs_identity ON orgs(identity);

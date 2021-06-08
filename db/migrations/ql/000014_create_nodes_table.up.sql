CREATE TABLE nodes (
  id             string          NOT NULL,  
  message_id     string          NOT NULL,
  owner          string          NOT NULL,
  identity       string          NOT NULL,
  description    string          NOT NULL,
  dx_peer        string,
  dx_endpoint    blob,
  created        int64           NOT NULL
);

CREATE UNIQUE INDEX nodes_id ON nodes(id);
CREATE UNIQUE INDEX nodes_identity ON nodes(identity);
CREATE UNIQUE INDEX nodes_owner ON nodes(owner);
CREATE UNIQUE INDEX nodes_peer ON nodes(dx_peer);

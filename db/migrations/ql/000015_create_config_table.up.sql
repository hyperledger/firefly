CREATE TABLE blocked (
  id             string           NOT NULL,  
  key            string           NOT NULL,
  value          string           NOT NULL,
);

CREATE UNIQUE INDEX blocked_id ON blocked(id);
CREATE UNIQUE INDEX blocked_key ON blocked(key);

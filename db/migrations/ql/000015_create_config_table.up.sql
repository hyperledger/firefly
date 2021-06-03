CREATE TABLE config (
  id             string           NOT NULL,  
  key            string           NOT NULL,
  value          string           NOT NULL,
);

CREATE UNIQUE INDEX config_id ON config(id);
CREATE UNIQUE INDEX config_key ON config(key);

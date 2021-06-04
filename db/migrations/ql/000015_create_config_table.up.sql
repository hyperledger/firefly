CREATE TABLE config (
  id             string           NOT NULL,  
  config_key     string           NOT NULL,
  config_value   string           NOT NULL,
);

CREATE UNIQUE INDEX config_id ON config(id);
CREATE UNIQUE INDEX config_config_key ON config(config_key);

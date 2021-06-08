CREATE TABLE config (
  config_key     string           NOT NULL,
  config_value   blob             NOT NULL
);

CREATE UNIQUE INDEX config_config_key ON config(config_key);

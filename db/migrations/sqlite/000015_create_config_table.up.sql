CREATE TABLE config (
  seq               INTEGER         PRIMARY KEY AUTOINCREMENT,
  config_key        VARCHAR(512)    NOT NULL,
  config_value      BYTEA           NOT NULL
);
CREATE UNIQUE INDEX config_sequence ON config(seq);
CREATE UNIQUE INDEX config_config_key ON config(config_key);


BEGIN;
CREATE TABLE config (
  seq               SERIAL          PRIMARY KEY,
  "key"             VARCHAR(512)    NOT NULL,
  "value"           BYTEA           NOT NULL
);
CREATE UNIQUE INDEX config_sequence ON config(seq);
CREATE UNIQUE INDEX config_key ON config("key");

COMMIT;
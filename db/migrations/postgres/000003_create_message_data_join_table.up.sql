BEGIN;
CREATE SEQUENCE messages_data_seq;
CREATE TABLE messages_data (
  seq        BIGINT   NOT NULL DEFAULT nextval('messages_data_seq'),
  message_id CHAR(36) NOT NULL REFERENCES messages(id),
  data_id    CHAR(36) NOT NULL REFERENCES data(id),
  data_hash  CHAR(64) NOT NULL,
  data_idx   INT      NOT NULL,
  PRIMARY KEY (message_id, data_id)
);
COMMIT;
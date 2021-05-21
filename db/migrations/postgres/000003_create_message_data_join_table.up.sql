BEGIN;
CREATE TABLE messages_data (
  message_id UUID     NOT NULL REFERENCES messages(id),
  data_id    UUID     NOT NULL REFERENCES data(id),
  data_hash  CHAR(64) NOT NULL,
  data_idx   INT      NOT NULL,
  PRIMARY KEY (message_id, data_id)
);
COMMIT;
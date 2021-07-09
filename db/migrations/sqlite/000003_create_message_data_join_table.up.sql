CREATE TABLE messages_data (
  seq        SERIAL   PRIMARY KEY,
  message_id UUID     NOT NULL,
  data_id    UUID     NOT NULL,
  data_hash  CHAR(64) NOT NULL,
  data_idx   INT      NOT NULL
);
CREATE UNIQUE INDEX messages_data_idx ON messages_data(message_id, data_id);

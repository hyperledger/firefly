CREATE TABLE messages_data (
  message_id string NOT NULL,
  data_id    string NOT NULL
);

CREATE UNIQUE INDEX messages_data_primary ON messages_data(message_id,data_id)
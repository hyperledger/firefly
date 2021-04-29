CREATE TABLE messages_data (
  message_id CHAR(36) NOT NULL REFERENCES messages(id),
  data_id    CHAR(36) NOT NULL REFERENCES data(id),
  PRIMARY KEY (message_id, data_id)
);
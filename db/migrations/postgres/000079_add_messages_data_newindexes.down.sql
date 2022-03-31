BEGIN;

DROP INDEX messages_data_message;
DROP INDEX messages_data_data;

CREATE UNIQUE INDEX messages_data_idx ON messages_data(message_id, data_id);

COMMIT;

BEGIN;

DROP INDEX messages_data_idx;

CREATE INDEX messages_data_message ON messages_data(message_id);
CREATE INDEX messages_data_data ON messages_data(data_id);

COMMIT;

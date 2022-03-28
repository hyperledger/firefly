BEGIN;

CREATE INDEX tokentransfer_messageid ON tokentransfer(message_id);

COMMIT;

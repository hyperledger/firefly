BEGIN;
DROP INDEX blockchainevents_subscription_id;
ALTER TABLE blockchainevents RENAME COLUMN subscription_id TO listener_id;
CREATE INDEX blockchainevents_listener_id ON blockchainevents(listener_id);
COMMIT;

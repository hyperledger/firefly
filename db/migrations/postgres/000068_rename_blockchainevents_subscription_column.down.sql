DROP INDEX blockchainevents_listener_id;
ALTER TABLE blockchainevents RENAME COLUMN listener_id TO subscription_id;
CREATE INDEX blockchainevents_subscription_id ON blockchainevents(subscription_id);
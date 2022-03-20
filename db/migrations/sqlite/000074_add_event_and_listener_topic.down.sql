DROP INDEX events_topic;

ALTER TABLE events DROP COLUMN topic;
ALTER TABLE contractlisteners DROP COLUMN topic;


DROP INDEX events_tag;
DROP INDEX events_topic;

ALTER TABLE events DROP COLUMN tag;
ALTER TABLE events DROP COLUMN topic;

BEGIN;

DROP INDEX events_topic;

ALTER TABLE events DROP COLUMN topic;

COMMIT;

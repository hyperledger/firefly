BEGIN;

ALTER TABLE events ADD COLUMN topic VARCHAR(64);

UPDATE events SET topic = '';

ALTER TABLE events ALTER COLUMN topic SET NOT NULL;

CREATE INDEX events_topic ON events(topic);

COMMIT;

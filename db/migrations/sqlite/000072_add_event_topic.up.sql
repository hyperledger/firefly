ALTER TABLE events ADD COLUMN topic VARCHAR(64);

UPDATE events SET topic = '';

CREATE INDEX events_topic ON events(topic);

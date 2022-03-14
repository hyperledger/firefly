ALTER TABLE events ADD COLUMN tag VARCHAR(64);
ALTER TABLE events ADD COLUMN topic VARCHAR(64);

UPDATE events set tag = '', topic = '';

CREATE INDEX events_tag ON events(tag);
CREATE INDEX events_topic ON events(topic);

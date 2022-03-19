ALTER TABLE events ADD COLUMN topic VARCHAR(64);
ALTER TABLE contractlisteners ADD COLUMN topic VARCHAR(64);

UPDATE events SET topic = '';
UPDATE contractlisteners SET topic = '';

CREATE INDEX events_topic ON events(topic);

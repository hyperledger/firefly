BEGIN;

ALTER TABLE events ADD COLUMN topic VARCHAR(64);
ALTER TABLE contractlisteners ADD COLUMN topic VARCHAR(64);

UPDATE events SET topic = '';
UPDATE contractlisteners SET topic = '';

ALTER TABLE events ALTER COLUMN topic SET NOT NULL;
ALTER TABLE contractlisteners ALTER COLUMN topic SET NOT NULL;

CREATE INDEX events_topic ON events(topic);

COMMIT;

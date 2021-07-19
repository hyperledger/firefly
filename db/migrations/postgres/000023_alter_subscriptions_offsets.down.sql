BEGIN;

ALTER TABLE subscriptions DROP COLUMN updated;

-- We change the primary key by which we access the data, so truncate the table
-- Meaning offsets for subscriptions will be reset going down
DELETE FROM offsets;

DROP INDEX offsets_unique;

ALTER TABLE offsets ADD id UUID NOT NULL;
ALTER TABLE offsets ADD namespace VARCHAR(64) NOT NULL;

CREATE UNIQUE INDEX offsets_id ON offsets(id);
CREATE UNIQUE INDEX offsets_unique ON offsets(otype,namespace,name);

COMMIT;

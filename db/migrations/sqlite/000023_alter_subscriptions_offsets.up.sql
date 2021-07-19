ALTER TABLE subscriptions ADD updated BIGINT;

-- We change the primary key by which we access the data, so truncate the table
-- Meaning offsets for subscriptions will be reset going up
DELETE FROM offsets;

DROP INDEX offsets_id;
DROP INDEX offsets_unique;

ALTER TABLE offsets DROP COLUMN namespace;
ALTER TABLE offsets DROP COLUMN id;

CREATE UNIQUE INDEX offsets_unique ON offsets(otype,name);

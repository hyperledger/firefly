ALTER TABLE subscriptions ADD updated BIGINT;

DROP INDEX offsets_id;
DROP INDEX offsets_unique;
ALTER TABLE offsets DROP COLUMN namespace;
ALTER TABLE offsets DROP COLUMN id;

CREATE UNIQUE INDEX offsets_unique ON offsets(otype,name);

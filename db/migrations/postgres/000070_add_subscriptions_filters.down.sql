BEGIN;
ALTER TABLE subscriptions DROP COLUMN filters;
ALTER TABLE subscriptions INSERT COLUMN filter_events text;
ALTER TABLE subscriptions INSERT COLUMN filter_topics text;
ALTER TABLE subscriptions INSERT COLUMN filter_tag text;
ALTER TABLE subscriptions INSERT COLUMN filter_group text;
COMMIT;
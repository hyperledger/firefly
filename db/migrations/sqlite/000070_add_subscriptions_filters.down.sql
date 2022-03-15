ALTER TABLE subscriptions DROP COLUMN filters;
ALTER TABLE subscriptions ADD COLUMN filter_events text;
ALTER TABLE subscriptions ADD COLUMN filter_topics text;
ALTER TABLE subscriptions ADD COLUMN filter_tag text;
ALTER TABLE subscriptions ADD COLUMN filter_group text;
BEGIN;
ALTER TABLE subscriptions ADD COLUMN filters TEXT;
ALTER TABLE subscriptions DROP COLUMN filter_events;
ALTER TABLE subscriptions DROP COLUMN filter_topics;
ALTER TABLE subscriptions DROP COLUMN filter_tag;
ALTER TABLE subscriptions DROP COLUMN filter_group;
COMMIT;

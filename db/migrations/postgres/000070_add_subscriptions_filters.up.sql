BEGIN;
ALTER TABLE subscriptions ADD COLUMN filters TEXT;
UPDATE subscriptions SET filters='{"events":"' || filter_events || '","message":{"topics":"' || filter_topics || '","tag":"' || filter_tag || '","group":"' || filter_group || '"}}';
ALTER TABLE subscriptions DROP COLUMN filter_events;
ALTER TABLE subscriptions DROP COLUMN filter_topics;
ALTER TABLE subscriptions DROP COLUMN filter_tag;
ALTER TABLE subscriptions DROP COLUMN filter_group;
COMMIT;
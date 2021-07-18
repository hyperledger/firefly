BEGIN;
ALTER TABLE subscriptions DROP COLUMN updated;
COMMIT;

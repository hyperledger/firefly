BEGIN;

ALTER TABLE subscriptions ADD updated BIGINT;

COMMIT;

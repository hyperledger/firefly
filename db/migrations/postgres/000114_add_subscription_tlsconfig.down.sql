BEGIN;
ALTER TABLE subscriptions DROP COLUMN tls_config_name;
COMMIT;
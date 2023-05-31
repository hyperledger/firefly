ALTER TABLE subscriptions ADD COLUMN tls_config_name VARCHAR(64);
UPDATE subscriptions SET tls_config_name = '';
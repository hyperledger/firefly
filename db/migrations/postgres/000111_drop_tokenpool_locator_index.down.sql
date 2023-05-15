BEGIN;
CREATE INDEX tokenpool_locator ON tokenpool(namespace, connector, locator);
COMMIT;

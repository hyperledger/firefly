BEGIN;
ALTER TABLE pins ADD COLUMN namespace VARCHAR(64);
UPDATE pins SET namespace = 'ff_system';
ALTER TABLE pins ALTER COLUMN namespace SET NOT NULL;

DROP INDEX pins_pin;
CREATE UNIQUE INDEX pins_pin ON pins(namespace, hash, batch_id, idx);
COMMIT;

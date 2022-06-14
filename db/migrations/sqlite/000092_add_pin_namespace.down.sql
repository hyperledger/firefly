DROP INDEX pins_pin;
CREATE UNIQUE INDEX pins_pin ON pins(hash, batch_id, idx);

ALTER TABLE pins DROP COLUMN namespace;

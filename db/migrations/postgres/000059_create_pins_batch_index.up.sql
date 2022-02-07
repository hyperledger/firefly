BEGIN;
CREATE INDEX pins_batch ON pins(batch_id);
COMMIT;
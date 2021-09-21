BEGIN;
ALTER TABLE operations RENAME COLUMN output TO info;
ALTER TABLE operations DROP COLUMN input;
COMMIT;

BEGIN;
ALTER TABLE tokenpool ADD COLUMN "key" VARCHAR(1024);
UPDATE tokenpool SET "key" = '';
COMMIT;

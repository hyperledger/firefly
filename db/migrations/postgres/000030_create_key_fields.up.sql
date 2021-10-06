BEGIN;

ALTER TABLE batches ADD "key" VARCHAR(1024);
UPDATE batches SET "key" = '';
ALTER TABLE batches ALTER COLUMN "key" SET NOT NULL;

ALTER TABLE messages ADD "key" VARCHAR(1024);
UPDATE messages SET "key" = '';
ALTER TABLE messages ALTER COLUMN "key" SET NOT NULL;

ALTER TABLE tokenpool ADD "key" VARCHAR(1024);
UPDATE tokenpool SET "key" = '';
ALTER TABLE tokenpool ALTER COLUMN "key" SET NOT NULL;

COMMIT;

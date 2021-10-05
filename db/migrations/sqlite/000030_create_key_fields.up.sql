ALTER TABLE batches ADD COLUMN "key" VARCHAR(1024);
UPDATE batches SET "key" = "";

ALTER TABLE messages ADD COLUMN "key" VARCHAR(1024);
UPDATE messages SET "key" = "";

ALTER TABLE tokenpool ADD COLUMN "key" VARCHAR(1024);
UPDATE tokenpool SET "key" = "";

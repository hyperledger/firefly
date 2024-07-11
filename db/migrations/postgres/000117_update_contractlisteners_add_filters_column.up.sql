BEGIN;
ALTER TABLE contractlisteners ADD COLUMN filters TEXT;
-- changing the length of varchar does not affect the index
ALTER TABLE contractlisteners ALTER COLUMN signature TYPE VARCHAR; 
COMMIT;
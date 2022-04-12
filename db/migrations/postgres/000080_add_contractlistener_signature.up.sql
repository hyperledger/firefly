BEGIN;
ALTER TABLE contractlisteners ADD COLUMN signature VARCHAR(1024);
CREATE INDEX contractlisteners_signature ON contractlisteners(signature);
COMMIT;

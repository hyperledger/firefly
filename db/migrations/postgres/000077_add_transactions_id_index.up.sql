BEGIN;

-- We created an invalid index in versions v0.14.1 and earlier - replace it if it exists here
-- For PostgreSQL we use an IF EXISTS here and removed the invalid creation from the earlier migration, for new envs
DROP INDEX transactions_id IF EXISTS;
CREATE UNIQUE INDEX transactions_id ON transactions(id);

COMMIT;

-- We created an invalid index in versions v0.14.1 and earlier - replace it if it exists here
-- We use an IF EXISTS here, as we no longer create the invalid index for new environments
DROP INDEX IF EXISTS transactions_id;

CREATE UNIQUE INDEX transactions_id ON transactions(id);

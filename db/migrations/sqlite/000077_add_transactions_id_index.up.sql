DROP INDEX transactions_id;
CREATE UNIQUE INDEX transactions_id ON transactions(id);

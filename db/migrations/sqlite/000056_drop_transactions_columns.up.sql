DROP INDEX transactions_protocol_id;
DROP INDEX transactions_ref;

ALTER TABLE transactions DROP COLUMN ref;
ALTER TABLE transactions DROP COLUMN signer;
ALTER TABLE transactions DROP COLUMN hash;
ALTER TABLE transactions DROP COLUMN protocol_id;
ALTER TABLE transactions DROP COLUMN info;

BEGIN;
DROP INDEX transactions_protocol_id;
DROP INDEX transactions_ref;

ALTER TABLE transactions DROP COLUMN ref;
ALTER TABLE transactions DROP COLUMN signer;
ALTER TABLE transactions DROP COLUMN hash;
ALTER TABLE transactions DROP COLUMN protocol_id;
ALTER TABLE transactions DROP COLUMN info;
ALTER TABLE transactions DROP COLUMN status;

ALTER TABLE transactions ADD COLUMN blockchain_ids VARCHAR(1024);
CREATE INDEX transactions_blockchain_ids ON transactions(blockchain_ids);
COMMIT;

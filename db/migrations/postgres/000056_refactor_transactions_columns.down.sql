BEGIN;
ALTER TABLE transactions ADD COLUMN ref         UUID;
ALTER TABLE transactions ADD COLUMN signer      VARCHAR(1024);
ALTER TABLE transactions ADD COLUMN hash        CHAR(64);
ALTER TABLE transactions ADD COLUMN protocol_id VARCHAR(256);
ALTER TABLE transactions ADD COLUMN info        TEXT;
ALTER TABLE transactions ADD COLUMN status      VARCHAR(64);

CREATE INDEX transactions_protocol_id ON transactions(protocol_id);
CREATE INDEX transactions_ref ON transactions(ref);

DROP INDEX transactions_blockchain_ids;
ALTER TABLE transactions DROP COLUMN blockchain_ids;
COMMIT;

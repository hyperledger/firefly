ALTER TABLE transactions ADD COLUMN ref         UUID;
ALTER TABLE transactions ADD COLUMN signer      VARCHAR(1024)   NOT NULL;
ALTER TABLE transactions ADD COLUMN hash        CHAR(64)        NOT NULL;
ALTER TABLE transactions ADD COLUMN protocol_id VARCHAR(256);
ALTER TABLE transactions ADD COLUMN info        BYTEA;

CREATE INDEX transactions_protocol_id ON transactions(protocol_id);
CREATE INDEX transactions_ref ON transactions(ref);

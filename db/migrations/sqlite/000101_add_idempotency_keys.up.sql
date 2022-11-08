ALTER TABLE transactions ADD COLUMN idempotency_key VARCHAR(256);
ALTER TABLE messages ADD COLUMN idempotency_key VARCHAR(256);

CREATE UNIQUE INDEX transactions_idempotency_keys ON transactions (namespace, idempotency_key);
CREATE UNIQUE INDEX messages_idempotency_keys ON messages (namespace, idempotency_key);

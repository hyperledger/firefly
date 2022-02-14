ALTER TABLE operations ADD COLUMN backend_id VARCHAR(256);
CREATE INDEX operations_backend ON operations(backend_id);

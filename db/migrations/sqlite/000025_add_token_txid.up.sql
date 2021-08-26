ALTER TABLE tokenpool ADD COLUMN tx_type  VARCHAR(64)  NOT NULL;
ALTER TABLE tokenpool ADD COLUMN tx_id    UUID;

CREATE INDEX tokenpool_fortx ON tokenpool(namespace,tx_id);

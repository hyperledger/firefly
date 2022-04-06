BEGIN;

DROP INDEX nonces_hash;

ALTER TABLE nonces RENAME COLUMN "hash" TO "context";
ALTER TABLE nonces ADD COLUMN group_hash CHAR(64);
ALTER TABLE nonces ADD COLUMN topic VARCHAR(64);

CREATE INDEX nonces_context ON nonces(context);
CREATE INDEX nonces_group ON nonces(group_hash);

COMMIT;

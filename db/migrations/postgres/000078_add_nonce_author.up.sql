BEGIN;

DROP INDEX nonces_context;
DROP INDEX nonces_group;

ALTER TABLE nonces RENAME COLUMN "context" TO "hash";
ALTER TABLE nonces DROP COLUMN "group_hash";
ALTER TABLE nonces DROP COLUMN "topic";

CREATE INDEX nonces_hash ON nonces(hash);

COMMIT;

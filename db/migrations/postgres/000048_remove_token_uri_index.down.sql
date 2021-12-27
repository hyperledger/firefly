BEGIN;
CREATE UNIQUE INDEX tokenbalance_uri ON tokenbalance(namespace,key,pool_id,uri);
COMMIT;

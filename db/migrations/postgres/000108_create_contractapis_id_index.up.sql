BEGIN;
DELETE FROM contractapis WHERE seq NOT IN (
  SELECT MAX(seq) FROM contractapis GROUP BY namespace, id);
CREATE UNIQUE INDEX contractapis_id ON contractapis(namespace,id);
COMMIT;

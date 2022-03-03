BEGIN;
ALTER TABLE contractsubscriptions RENAME TO contractlisteners;
COMMIT;
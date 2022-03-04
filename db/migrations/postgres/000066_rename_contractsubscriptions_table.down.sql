BEGIN;
ALTER TABLE contractlisteners RENAME TO contractsubscriptions;
COMMIT;
BEGIN;
ALTER TABLE identities RENAME COLUMN messages_claim TO messages_claim_old;
ALTER TABLE identities ADD COLUMN messages_claim UUID;
UPDATE identities SET messages_claim = messages_claim_old;
ALTER TABLE identities DROP COLUMN messages_claim_old;
COMMIT;

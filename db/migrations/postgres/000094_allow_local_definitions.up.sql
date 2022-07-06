BEGIN;
ALTER TABLE identities RENAME COLUMN messages_claim TO messages_claim_old;
ALTER TABLE identities ADD COLUMN messages_claim UUID;
UPDATE identities SET messages_claim = messages_claim_old;
ALTER TABLE identities DROP COLUMN messages_claim_old;

ALTER TABLE ffi RENAME COLUMN message_id TO message_id_old;
ALTER TABLE ffi ADD COLUMN message_id UUID;
UPDATE ffi SET message_id = message_id_old;
ALTER TABLE ffi DROP COLUMN message_id_old;

ALTER TABLE contractapis RENAME COLUMN message_id TO message_id_old;
ALTER TABLE contractapis ADD COLUMN message_id UUID;
UPDATE contractapis SET message_id = message_id_old;
ALTER TABLE contractapis DROP COLUMN message_id_old;
COMMIT;

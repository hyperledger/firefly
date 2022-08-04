DROP INDEX identities_id;
CREATE UNIQUE INDEX identities_id ON identities(id);

DROP INDEX verifiers_hash;
CREATE UNIQUE INDEX verifiers_hash on verifiers(hash);

DROP INDEX verifiers_identity;
CREATE UNIQUE INDEX verifiers_identity on verifiers(identity);

DROP INDEX tokenpool_locator;
CREATE UNIQUE INDEX tokenpool_locator ON tokenpool(connector, locator);

DROP INDEX ffi_id;
CREATE UNIQUE INDEX ffi_id ON ffi(id);

DROP INDEX datatypes_id;
CREATE UNIQUE INDEX datatypes_id ON datatypes(id);

DROP INDEX batches_id;
CREATE UNIQUE INDEX batches_id ON batches(id);

DROP INDEX groups_hash;
CREATE UNIQUE INDEX groups_hash ON groups(hash);

DROP INDEX messages_id;
CREATE UNIQUE INDEX messages_id ON messages(id);

DROP INDEX data_id;
CREATE UNIQUE INDEX data_id ON data(id);

ALTER TABLE messages_data RENAME TO messages_data_old;
CREATE TABLE messages_data (
  seq         SERIAL          PRIMARY KEY,
  message_id  UUID            NOT NULL,
  data_id     UUID            NOT NULL,
  data_hash   CHAR(64)        NOT NULL,
  data_idx    INT             NOT NULL
);
INSERT INTO messages_data(message_id, data_id, data_hash, data_idx)
  SELECT message_id, data_id, data_hash, data_idx FROM messages_data_old;
DROP TABLE messages_data_old;

CREATE INDEX messages_data_message ON messages_data(message_id);
CREATE INDEX messages_data_data ON messages_data(data_id);

DROP INDEX nextpins_context;
ALTER TABLE nextpins DROP COLUMN namespace;
CREATE INDEX nextpins_hash ON nextpins(hash);

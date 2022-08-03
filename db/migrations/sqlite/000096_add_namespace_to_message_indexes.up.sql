DROP INDEX identities_id;
CREATE UNIQUE INDEX identities_id ON identities(namespace, id);

DROP INDEX verifiers_hash;
CREATE UNIQUE INDEX verifiers_hash on verifiers(namespace, hash);

DROP INDEX verifiers_identity;
CREATE UNIQUE INDEX verifiers_identity on verifiers(namespace, identity);

DROP INDEX tokenpool_locator;
CREATE UNIQUE INDEX tokenpool_locator ON tokenpool(namespace, connector, locator);

DROP INDEX ffi_id;
CREATE UNIQUE INDEX ffi_id ON ffi(namespace, id);

DROP INDEX datatypes_id;
CREATE UNIQUE INDEX datatypes_id ON datatypes(namespace, id);

DROP INDEX batches_id;
CREATE UNIQUE INDEX batches_id ON batches(namespace, id);

DROP INDEX groups_hash;
CREATE UNIQUE INDEX groups_hash ON groups(namespace_local, hash);

DROP INDEX messages_id;
CREATE UNIQUE INDEX messages_id ON messages(namespace_local, id);

DROP INDEX data_id;
CREATE UNIQUE INDEX data_id ON data(namespace, id);

ALTER TABLE messages_data RENAME TO messages_data_old;
CREATE TABLE messages_data (
  seq         INTEGER         PRIMARY KEY AUTOINCREMENT,
  message_id  UUID            NOT NULL,
  data_id     UUID            NOT NULL,
  data_hash   CHAR(64)        NOT NULL,
  data_idx    INT             NOT NULL,
  namespace   VARCHAR(64)
);
INSERT INTO messages_data(message_id, data_id, data_hash, data_idx)
  SELECT message_id, data_id, data_hash, data_idx FROM messages_data_old;
DROP TABLE messages_data_old;
UPDATE messages_data SET namespace = msg.namespace
  FROM (SELECT namespace, id FROM messages) AS msg
  WHERE messages_data.message_id = msg.id;

CREATE INDEX messages_data_message ON messages_data(namespace, message_id);
CREATE INDEX messages_data_data ON messages_data(namespace, data_id);

ALTER TABLE nextpins ADD COLUMN namespace VARCHAR(64);
DROP INDEX nextpins_hash;
CREATE INDEX nextpins_context ON nextpins(namespace, context);

CREATE TABLE contractsubscriptions (
  seq              INTEGER         PRIMARY KEY AUTOINCREMENT,
  id               UUID            NOT NULL,
  interface_id     UUID            NULL,
  event_id         UUID            NOT NULL,
  namespace        VARCHAR(64)     NOT NULL,
  protocol_id      VARCHAR(1024)   NOT NULL,
  location         BYTEA           NOT NULL
);

CREATE UNIQUE INDEX contractsubscriptions_protocolid ON contractsubscriptions(protocol_id);

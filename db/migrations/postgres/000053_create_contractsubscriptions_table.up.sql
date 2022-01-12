BEGIN;
CREATE TABLE contractsubscriptions (
  seq              SERIAL          PRIMARY KEY,
  id               UUID            NOT NULL,
  interface_id     UUID            NULL,
  event            TEXT            NOT NULL,
  namespace        VARCHAR(64)     NOT NULL,
  name             VARCHAR(64)     NULL,
  protocol_id      VARCHAR(1024)   NOT NULL,
  location         TEXT            NOT NULL,
  created          BIGINT          NOT NULL
);

CREATE UNIQUE INDEX contractsubscriptions_protocolid ON contractsubscriptions(protocol_id);
CREATE UNIQUE INDEX contractsubscriptions_name ON contractsubscriptions(namespace,name);
COMMIT;
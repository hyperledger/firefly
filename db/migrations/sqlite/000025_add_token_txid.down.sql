CREATE TABLE tokenpool_new (
  seq            INTEGER         PRIMARY KEY AUTOINCREMENT,
  id             UUID            NOT NULL,
  namespace      VARCHAR(64)     NOT NULL,
  name           VARCHAR(64)     NOT NULL,
  protocol_id    VARCHAR(1024)   NOT NULL,
  type           VARCHAR(64)     NOT NULL
);

INSERT INTO tokenpool_new SELECT seq,id,namespace,name,protocol_id,type FROM tokenpool;
DROP TABLE tokenpool;
ALTER TABLE tokenpool_new RENAME TO tokenpool;

CREATE UNIQUE INDEX tokenpool_id ON tokenpool(id);
CREATE UNIQUE INDEX tokenpool_name ON tokenpool(namespace,name);

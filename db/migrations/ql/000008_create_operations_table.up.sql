CREATE TABLE operations (
  id          string         NOT NULL,
  namespace   string         NOT NULL,
  msg_id      string,
  data_id     string,
  optype      string         NOT NULL,
  opdir       string         NOT NULL,
  opstatus    string         NOT NULL,
  recipient   string,
  plugin      string         NOT NULL,
  backend_id   string         NOT NULL,
  created     int64          NOT NULL,
  updated     int64          NOT NULL,
  error       string         NOT NULL,
);

CREATE UNIQUE INDEX operations_primary ON operations(id);
CREATE INDEX operations_search ON operations(namespace,msg_id,optype,opdir,opstatus,error);

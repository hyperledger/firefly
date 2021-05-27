CREATE TABLE blocked (
  id             string           NOT NULL,  
  namespace      string           NOT NULL,
  context        string           NOT NULL,
  group_id       string,
  message_id     string           NOT NULL,
  created        int64            NOT NULL
);

CREATE UNIQUE INDEX blocked_id ON blocked(id);
CREATE UNIQUE INDEX blocked_message ON blocked(message_id);
CREATE UNIQUE INDEX blocked_context ON blocked(namespace,context,group_id);

BEGIN;
CREATE INDEX messages_topics_tag ON messages(namespace,topics,tag);
COMMIT;

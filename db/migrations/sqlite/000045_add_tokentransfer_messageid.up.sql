ALTER TABLE tokentransfer ADD COLUMN message_id UUID;

UPDATE tokentransfer SET message_id = message.id
  FROM (SELECT hash, id FROM messages) AS message
  WHERE tokentransfer.message_hash = message.hash;

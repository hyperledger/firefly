ALTER TABLE tokenapproval ADD COLUMN message_id UUID;
ALTER TABLE tokenapproval ADD COLUMN message_hash CHAR(64);
CREATE INDEX tokenapproval_messageid ON tokenapproval(message_id);
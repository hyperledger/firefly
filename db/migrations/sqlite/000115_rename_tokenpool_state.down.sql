ALTER TABLE tokenpool ADD COLUMN state VARCHAR(64);
UPDATE tokenpool SET state = (CASE WHEN active = true THEN 'confirmed' ELSE 'pending' END);
ALTER TABLE tokenpool DROP COLUMN active;

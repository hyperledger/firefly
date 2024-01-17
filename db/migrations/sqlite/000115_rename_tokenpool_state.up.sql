ALTER TABLE tokenpool ADD COLUMN active BOOLEAN;
UPDATE tokenpool SET active = (CASE WHEN state = 'confirmed' THEN true ELSE false END);
ALTER TABLE tokenpool DROP COLUMN state;

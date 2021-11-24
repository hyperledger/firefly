ALTER TABLE tokenpool ADD COLUMN state VARCHAR(64);
UPDATE tokenpool SET state="unknown";

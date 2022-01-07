ALTER TABLE operations RENAME COLUMN info TO output;
ALTER TABLE operations ADD COLUMN input TEXT;

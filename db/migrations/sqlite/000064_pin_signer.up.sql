ALTER TABLE pins ADD COLUMN signer TEXT;
UPDATE pins SET signer = "";

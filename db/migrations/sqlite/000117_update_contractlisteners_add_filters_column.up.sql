ALTER TABLE contractlisteners ADD COLUMN filters TEXT;
-- in SQLITE VARCHAR is equivalent to TEXT so no migration for signature length
DROP INDEX tokenapproval_subject;
CREATE INDEX tokenapproval_subject ON tokenapproval(pool_id, subject);

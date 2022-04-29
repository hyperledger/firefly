DROP INDEX tokenapproval_subject;
CREATE UNIQUE INDEX tokenapproval_subject ON tokenapproval(pool_id, subject);

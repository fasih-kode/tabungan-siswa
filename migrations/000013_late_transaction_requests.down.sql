ALTER TABLE audit_logs DROP CONSTRAINT audit_logs_entity_type_check;
ALTER TABLE audit_logs ADD CONSTRAINT audit_logs_entity_type_check CHECK (entity_type IN (
    'USER','ACADEMIC_YEAR','CLASS','STUDENT','STUDENT_CLASS_HISTORY',
    'TEACHER_CLASS_ASSIGNMENT','CLASS_TRANSFER_REQUEST','SAVINGS_ACCOUNT',
    'TRANSACTION','SAVINGS_SETTLEMENT','SETTLEMENT_REOPEN_REQUEST'
));
ALTER TABLE transactions DROP COLUMN late_request_id;
DROP INDEX IF EXISTS leaving_transaction_requests_active_evidence_idx;
DROP TABLE leaving_transaction_requests;
ALTER TABLE students DROP COLUMN leaving_date;

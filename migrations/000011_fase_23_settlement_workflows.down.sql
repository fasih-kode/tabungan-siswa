DROP TABLE IF EXISTS settlement_reopen_requests;
DROP TABLE IF EXISTS settlement_cash_movements;

CREATE OR REPLACE FUNCTION enforce_academic_year_create_after_previous_closed()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM academic_years WHERE status <> 'CLOSED') THEN
        RAISE EXCEPTION 'new academic year requires all previous years to be CLOSED'
            USING ERRCODE = '23514', CONSTRAINT = 'academic_years_create_requires_closed_previous';
    END IF;
    RETURN NEW;
END;
$$;

ALTER TABLE audit_logs DROP CONSTRAINT audit_logs_entity_type_check;
ALTER TABLE audit_logs ADD CONSTRAINT audit_logs_entity_type_check CHECK (entity_type IN (
    'USER','ACADEMIC_YEAR','CLASS','STUDENT','STUDENT_CLASS_HISTORY',
    'TEACHER_CLASS_ASSIGNMENT','CLASS_TRANSFER_REQUEST','SAVINGS_ACCOUNT',
    'TRANSACTION','SAVINGS_SETTLEMENT'
));

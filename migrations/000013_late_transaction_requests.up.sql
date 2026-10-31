ALTER TABLE students ADD COLUMN leaving_date DATE;

CREATE TABLE leaving_transaction_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    savings_account_id UUID NOT NULL REFERENCES savings_accounts(id),
    academic_year_id UUID NOT NULL REFERENCES academic_years(id),
    transaction_type TEXT NOT NULL CHECK (transaction_type IN ('DEPOSIT','WITHDRAWAL')),
    amount NUMERIC(15,2) NOT NULL CHECK (amount > 0),
    transaction_date DATE NOT NULL,
    reason TEXT NOT NULL CHECK (char_length(btrim(reason)) BETWEEN 1 AND 500),
    evidence_reference TEXT NOT NULL CHECK (char_length(btrim(evidence_reference)) BETWEEN 1 AND 200),
    requested_by UUID NOT NULL REFERENCES users(id),
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL CHECK (status IN ('PENDING','APPROVED','REJECTED','COMPLETED')),
    decided_by UUID REFERENCES users(id),
    decided_at TIMESTAMPTZ,
    decision_reason TEXT,
    completed_at TIMESTAMPTZ,
    CONSTRAINT leaving_transaction_requests_state_check CHECK (
      (status='PENDING' AND decided_by IS NULL AND decided_at IS NULL AND decision_reason IS NULL AND completed_at IS NULL)
      OR (status='APPROVED' AND decided_by IS NOT NULL AND decided_at IS NOT NULL AND decision_reason IS NOT NULL AND completed_at IS NULL)
      OR (status='REJECTED' AND decided_by IS NOT NULL AND decided_at IS NOT NULL AND decision_reason IS NOT NULL AND completed_at IS NULL)
      OR (status='COMPLETED' AND decided_by IS NOT NULL AND decided_at IS NOT NULL AND decision_reason IS NOT NULL AND completed_at IS NOT NULL)
    )
);
ALTER TABLE transactions ADD COLUMN late_request_id UUID UNIQUE REFERENCES leaving_transaction_requests(id);
CREATE INDEX leaving_transaction_requests_year_status_idx ON leaving_transaction_requests(academic_year_id,status);
CREATE INDEX leaving_transaction_requests_account_status_idx ON leaving_transaction_requests(savings_account_id,status);
CREATE UNIQUE INDEX leaving_transaction_requests_active_evidence_idx ON leaving_transaction_requests(savings_account_id,lower(evidence_reference)) WHERE status IN ('PENDING','APPROVED');

ALTER TABLE audit_logs DROP CONSTRAINT audit_logs_entity_type_check;
ALTER TABLE audit_logs ADD CONSTRAINT audit_logs_entity_type_check CHECK (entity_type IN (
    'USER','ACADEMIC_YEAR','CLASS','STUDENT','STUDENT_CLASS_HISTORY',
    'TEACHER_CLASS_ASSIGNMENT','CLASS_TRANSFER_REQUEST','SAVINGS_ACCOUNT',
    'TRANSACTION','SAVINGS_SETTLEMENT','SETTLEMENT_REOPEN_REQUEST','LEAVING_TRANSACTION_REQUEST'
));

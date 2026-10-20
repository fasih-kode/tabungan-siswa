CREATE TABLE settlement_cash_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    savings_settlement_id UUID NOT NULL REFERENCES savings_settlements(id),
    kind TEXT NOT NULL,
    amount NUMERIC(15, 2) NOT NULL,
    confirmed_by UUID NOT NULL REFERENCES users(id),
    confirmed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    receipt_reference TEXT,
    CONSTRAINT settlement_cash_movements_kind_check CHECK (kind IN ('HANDOVER', 'RETURN')),
    CONSTRAINT settlement_cash_movements_amount_check CHECK (amount > 0),
    CONSTRAINT settlement_cash_movements_reference_length_check CHECK (receipt_reference IS NULL OR char_length(receipt_reference) <= 200)
);
CREATE INDEX settlement_cash_movements_settlement_idx
    ON settlement_cash_movements (savings_settlement_id, confirmed_at, id);

CREATE TABLE settlement_reopen_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    savings_settlement_id UUID NOT NULL REFERENCES savings_settlements(id),
    savings_account_id UUID NOT NULL REFERENCES savings_accounts(id),
    academic_year_id UUID NOT NULL REFERENCES academic_years(id),
    requested_by UUID NOT NULL REFERENCES users(id),
    reason TEXT NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL,
    decided_by UUID REFERENCES users(id),
    decided_at TIMESTAMPTZ,
    decision_reason TEXT,
    completed_at TIMESTAMPTZ,
    CONSTRAINT settlement_reopen_requests_reason_check CHECK (char_length(btrim(reason)) BETWEEN 1 AND 500),
    CONSTRAINT settlement_reopen_requests_status_check CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED', 'COMPLETED')),
    CONSTRAINT settlement_reopen_requests_decision_check CHECK (
        (status = 'PENDING' AND decided_by IS NULL AND decided_at IS NULL AND decision_reason IS NULL AND completed_at IS NULL)
        OR (status = 'APPROVED' AND decided_by IS NOT NULL AND decided_at IS NOT NULL AND decision_reason IS NOT NULL AND completed_at IS NULL)
        OR (status = 'REJECTED' AND decided_by IS NOT NULL AND decided_at IS NOT NULL AND decision_reason IS NOT NULL AND completed_at IS NULL)
        OR (status = 'COMPLETED' AND decided_by IS NOT NULL AND decided_at IS NOT NULL AND decision_reason IS NOT NULL AND completed_at IS NOT NULL)
    )
);
CREATE UNIQUE INDEX settlement_reopen_requests_one_active_per_settlement_idx
    ON settlement_reopen_requests (savings_settlement_id)
    WHERE status IN ('PENDING', 'APPROVED');
CREATE INDEX settlement_reopen_requests_year_status_idx
    ON settlement_reopen_requests (academic_year_id, status);

CREATE OR REPLACE FUNCTION enforce_academic_year_create_after_previous_closed()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM id FROM academic_years ORDER BY id FOR UPDATE;
    IF EXISTS (SELECT 1 FROM academic_years WHERE status <> 'CLOSED') THEN
        RAISE EXCEPTION 'new academic year requires all previous years to be CLOSED'
            USING ERRCODE = '23514', CONSTRAINT = 'academic_years_create_requires_closed_previous';
    END IF;
    IF EXISTS (SELECT 1 FROM settlement_reopen_requests WHERE status IN ('PENDING','APPROVED')) THEN
        RAISE EXCEPTION 'new academic year blocked by active settlement workflow'
            USING ERRCODE = '23514', CONSTRAINT = 'academic_years_create_requires_no_active_settlement_workflow';
    END IF;
    RETURN NEW;
END;
$$;

ALTER TABLE audit_logs DROP CONSTRAINT audit_logs_entity_type_check;
ALTER TABLE audit_logs ADD CONSTRAINT audit_logs_entity_type_check CHECK (entity_type IN (
    'USER','ACADEMIC_YEAR','CLASS','STUDENT','STUDENT_CLASS_HISTORY',
    'TEACHER_CLASS_ASSIGNMENT','CLASS_TRANSFER_REQUEST','SAVINGS_ACCOUNT',
    'TRANSACTION','SAVINGS_SETTLEMENT','SETTLEMENT_REOPEN_REQUEST'
));

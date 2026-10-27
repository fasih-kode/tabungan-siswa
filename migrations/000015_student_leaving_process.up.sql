CREATE TABLE student_leaving_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id),
    academic_year_id UUID NOT NULL REFERENCES academic_years(id),
    effective_leaving_date DATE NOT NULL,
    status TEXT NOT NULL,
    initiated_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finalized_by UUID REFERENCES users(id),
    finalized_at TIMESTAMPTZ,
    cancelled_by UUID REFERENCES users(id),
    cancelled_at TIMESTAMPTZ,
    cancellation_reason TEXT,

    CONSTRAINT student_leaving_requests_status_check
        CHECK (status IN ('IN_PROGRESS', 'FINALIZED', 'CANCELLED')),
    CONSTRAINT student_leaving_requests_state_metadata_check
        CHECK (
            (
                status = 'IN_PROGRESS'
                AND finalized_by IS NULL
                AND finalized_at IS NULL
                AND cancelled_by IS NULL
                AND cancelled_at IS NULL
                AND cancellation_reason IS NULL
            )
            OR (
                status = 'FINALIZED'
                AND finalized_by IS NOT NULL
                AND finalized_at IS NOT NULL
                AND cancelled_by IS NULL
                AND cancelled_at IS NULL
                AND cancellation_reason IS NULL
            )
            OR (
                status = 'CANCELLED'
                AND finalized_by IS NULL
                AND finalized_at IS NULL
                AND cancelled_by IS NOT NULL
                AND cancelled_at IS NOT NULL
                AND cancellation_reason IS NOT NULL
                AND char_length(btrim(cancellation_reason)) BETWEEN 1 AND 500
            )
        )
);

CREATE UNIQUE INDEX student_leaving_requests_one_in_progress_per_student_idx
    ON student_leaving_requests (student_id)
    WHERE status = 'IN_PROGRESS';

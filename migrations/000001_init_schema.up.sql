CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT users_role_check
        CHECK (role IN ('ADMIN', 'WALI_KELAS', 'SISWA'))
);

CREATE TABLE academic_years (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT academic_years_date_check
        CHECK (start_date < end_date),

    CONSTRAINT academic_years_status_check
        CHECK (status IN ('OPEN', 'CLOSING', 'CLOSED'))
);

CREATE UNIQUE INDEX academic_years_one_open_idx
    ON academic_years (status)
    WHERE status = 'OPEN';

CREATE TABLE classes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    level SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT classes_level_check
        CHECK (level BETWEEN 1 AND 12),

    CONSTRAINT classes_name_level_unique
        UNIQUE (name, level)
);

CREATE TABLE students (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE REFERENCES users(id),
    nis TEXT UNIQUE,
    nisn TEXT UNIQUE,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT students_status_check
        CHECK (status IN ('ACTIVE', 'LEFT'))
);

CREATE TABLE student_class_histories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id),
    academic_year_id UUID NOT NULL REFERENCES academic_years(id),
    class_id UUID NOT NULL REFERENCES classes(id),
    validity DATERANGE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT student_class_histories_validity_check
        CHECK (NOT isempty(validity)),

    CONSTRAINT student_class_histories_no_overlap
        EXCLUDE USING GIST (
            student_id WITH =,
            academic_year_id WITH =,
            validity WITH &&
        )
);

CREATE TABLE teacher_class_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    academic_year_id UUID NOT NULL REFERENCES academic_years(id),
    class_id UUID NOT NULL REFERENCES classes(id),
    validity DATERANGE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT teacher_class_assignments_validity_check
        CHECK (NOT isempty(validity)),

    CONSTRAINT teacher_class_assignments_no_overlap
        EXCLUDE USING GIST (
            academic_year_id WITH =,
            class_id WITH =,
            validity WITH &&
        )
);

CREATE TABLE class_transfer_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id),
    academic_year_id UUID NOT NULL REFERENCES academic_years(id),
    from_class_id UUID NOT NULL REFERENCES classes(id),
    to_class_id UUID NOT NULL REFERENCES classes(id),
    requested_by UUID NOT NULL REFERENCES users(id),
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL,
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,

    CONSTRAINT class_transfer_requests_class_check
        CHECK (from_class_id <> to_class_id),

    CONSTRAINT class_transfer_requests_status_check
        CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED')),

    CONSTRAINT class_transfer_requests_review_check
        CHECK (
            (status = 'PENDING' AND reviewed_by IS NULL AND reviewed_at IS NULL)
            OR
            (status IN ('APPROVED', 'REJECTED')
                AND reviewed_by IS NOT NULL
                AND reviewed_at IS NOT NULL)
        )
);

CREATE TABLE savings_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id),
    academic_year_id UUID NOT NULL REFERENCES academic_years(id),
    status TEXT NOT NULL,
    settled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT savings_accounts_student_year_unique
        UNIQUE (student_id, academic_year_id),

    CONSTRAINT savings_accounts_status_check
        CHECK (status IN ('OPEN', 'SETTLED')),

    CONSTRAINT savings_accounts_settled_at_check
        CHECK (
            (status = 'OPEN' AND settled_at IS NULL)
            OR
            (status = 'SETTLED' AND settled_at IS NOT NULL)
        )
);

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    savings_account_id UUID NOT NULL REFERENCES savings_accounts(id),
    type TEXT NOT NULL,
    amount NUMERIC(15, 2) NOT NULL,
    transaction_date DATE NOT NULL,
    status TEXT NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT transactions_type_check
        CHECK (type IN ('DEPOSIT', 'WITHDRAWAL')),

    CONSTRAINT transactions_amount_check
        CHECK (amount > 0),

    CONSTRAINT transactions_status_check
        CHECK (status IN ('ACTIVE', 'CANCELLED'))
);

CREATE TABLE savings_settlements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    savings_account_id UUID NOT NULL REFERENCES savings_accounts(id),
    type TEXT NOT NULL,
    amount NUMERIC(15, 2) NOT NULL,
    executed_by UUID NOT NULL REFERENCES users(id),
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL,
    replaces_settlement_id UUID REFERENCES savings_settlements(id),

    CONSTRAINT savings_settlements_type_check
        CHECK (type IN ('REGULAR_YEAR_END', 'STUDENT_LEAVING')),

    CONSTRAINT savings_settlements_amount_check
        CHECK (amount >= 0),

    CONSTRAINT savings_settlements_status_check
        CHECK (status IN ('COMPLETED', 'SUPERSEDED'))
);

CREATE UNIQUE INDEX savings_settlements_one_completed_idx
    ON savings_settlements (savings_account_id)
    WHERE status = 'COMPLETED';

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID REFERENCES users(id),
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    before_data JSONB,
    after_data JSONB,

    CONSTRAINT audit_logs_entity_type_check
        CHECK (
            entity_type IN (
                'USER',
                'ACADEMIC_YEAR',
                'CLASS',
                'STUDENT',
                'STUDENT_CLASS_HISTORY',
                'TEACHER_CLASS_ASSIGNMENT',
                'CLASS_TRANSFER_REQUEST',
                'SAVINGS_ACCOUNT',
                'TRANSACTION',
                'SAVINGS_SETTLEMENT'
            )
        )
);

CREATE INDEX student_class_histories_student_year_idx
    ON student_class_histories (student_id, academic_year_id);

CREATE INDEX teacher_class_assignments_user_year_idx
    ON teacher_class_assignments (user_id, academic_year_id);

CREATE INDEX class_transfer_requests_student_year_idx
    ON class_transfer_requests (student_id, academic_year_id);

CREATE INDEX class_transfer_requests_status_idx
    ON class_transfer_requests (status);

CREATE INDEX savings_accounts_student_idx
    ON savings_accounts (student_id);

CREATE INDEX savings_accounts_academic_year_idx
    ON savings_accounts (academic_year_id);

CREATE INDEX transactions_account_date_idx
    ON transactions (savings_account_id, transaction_date);

CREATE INDEX transactions_created_by_idx
    ON transactions (created_by);

CREATE INDEX savings_settlements_account_idx
    ON savings_settlements (savings_account_id);

CREATE INDEX audit_logs_entity_idx
    ON audit_logs (entity_type, entity_id);

CREATE INDEX audit_logs_actor_idx
    ON audit_logs (actor_user_id);

CREATE INDEX audit_logs_occurred_at_idx
    ON audit_logs (occurred_at);

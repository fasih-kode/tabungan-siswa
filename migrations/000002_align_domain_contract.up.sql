ALTER TABLE student_class_histories
    ADD COLUMN updated_at TIMESTAMPTZ;

UPDATE student_class_histories
SET updated_at = created_at;

ALTER TABLE student_class_histories
    ALTER COLUMN updated_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at SET NOT NULL;


ALTER TABLE teacher_class_assignments
    ADD COLUMN updated_at TIMESTAMPTZ;

UPDATE teacher_class_assignments
SET updated_at = created_at;

ALTER TABLE teacher_class_assignments
    ALTER COLUMN updated_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at SET NOT NULL;


ALTER TABLE class_transfer_requests
    ADD COLUMN updated_at TIMESTAMPTZ,
    ADD COLUMN rejection_reason TEXT;

UPDATE class_transfer_requests
SET updated_at = requested_at;

ALTER TABLE class_transfer_requests
    ALTER COLUMN updated_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at SET NOT NULL;


ALTER TABLE transactions
    ADD CONSTRAINT transactions_amount_integer_check
    CHECK (amount = trunc(amount, 0));


ALTER TABLE savings_settlements
    ADD CONSTRAINT savings_settlements_amount_integer_check
    CHECK (amount = trunc(amount, 0));

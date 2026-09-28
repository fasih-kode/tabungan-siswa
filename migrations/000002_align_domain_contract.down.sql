ALTER TABLE savings_settlements
    DROP CONSTRAINT IF EXISTS savings_settlements_amount_integer_check;

ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS transactions_amount_integer_check;

ALTER TABLE class_transfer_requests
    DROP COLUMN IF EXISTS rejection_reason,
    DROP COLUMN IF EXISTS updated_at;

ALTER TABLE teacher_class_assignments
    DROP COLUMN IF EXISTS updated_at;

ALTER TABLE student_class_histories
    DROP COLUMN IF EXISTS updated_at;

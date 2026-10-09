BEGIN;

DROP TRIGGER IF EXISTS transactions_cancel_requires_open_account_trigger ON transactions;
DROP FUNCTION IF EXISTS enforce_transaction_cancel_requires_open_account();

ALTER TABLE class_transfer_requests
    DROP CONSTRAINT IF EXISTS class_transfer_requests_rejection_reason_check;

DROP TRIGGER IF EXISTS class_transfer_open_academic_year_trigger ON class_transfer_requests;
DROP FUNCTION IF EXISTS enforce_class_transfer_open_academic_year();
DROP INDEX IF EXISTS class_transfer_requests_one_pending_per_student_year_idx;

DROP TRIGGER IF EXISTS savings_accounts_no_reopen_after_year_close_trigger ON savings_accounts;
DROP FUNCTION IF EXISTS enforce_savings_account_no_reopen_after_year_close();
DROP TRIGGER IF EXISTS savings_accounts_creation_eligibility_trigger ON savings_accounts;
DROP FUNCTION IF EXISTS enforce_savings_account_creation_eligibility();

DROP TRIGGER IF EXISTS academic_years_close_requires_settled_accounts_trigger ON academic_years;
DROP FUNCTION IF EXISTS enforce_academic_year_close_requires_settled_accounts();

DROP TRIGGER IF EXISTS academic_years_history_bounds_trigger ON academic_years;
DROP FUNCTION IF EXISTS enforce_academic_year_history_bounds();
DROP TRIGGER IF EXISTS student_class_histories_academic_year_bounds_trigger ON student_class_histories;
DROP FUNCTION IF EXISTS enforce_student_class_history_academic_year_bounds();

-- Restore legacy open-ended ranges only when the history was not changed after migration.
UPDATE student_class_histories h
   SET validity = b.original_validity,
       updated_at = b.original_updated_at
  FROM migration_000006_student_class_history_validity_backup b
 WHERE h.id = b.student_class_history_id
   AND h.validity = b.normalized_validity
   AND h.updated_at = b.normalized_updated_at;

-- Remove backup rows whose restoration completed. Preserve unresolved originals
-- when a history changed or was deleted after migration, rather than destroying
-- the only copy of its pre-migration range during rollback.
DELETE FROM migration_000006_student_class_history_validity_backup b
 WHERE EXISTS (
    SELECT 1
      FROM student_class_histories h
     WHERE h.id = b.student_class_history_id
       AND h.validity = b.original_validity
       AND h.updated_at = b.original_updated_at
 );

DO $$
DECLARE
    unresolved_rows BIGINT;
BEGIN
    SELECT COUNT(*)
      INTO unresolved_rows
      FROM migration_000006_student_class_history_validity_backup;

    IF unresolved_rows = 0 THEN
        DROP TABLE migration_000006_student_class_history_validity_backup;
    ELSE
        RAISE NOTICE
            'Rollback preserved migration_000006_student_class_history_validity_backup with % unresolved row(s); review before removing it',
            unresolved_rows;
    END IF;
END;
$$;

COMMIT;

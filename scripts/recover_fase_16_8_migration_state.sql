-- FASE 16.8 recovery: reconcile migration metadata only after validating existing objects.
-- This script does NOT create/drop triggers, functions, constraints, indexes, alter app rows,
-- or delete the backup table.
BEGIN;
DO $recovery$
DECLARE
    current_version BIGINT;
    current_dirty BOOLEAN;
    missing_objects TEXT;
    backup_rows BIGINT;
    index_definition TEXT;
    rejection_definition TEXT;
    overlap_definition TEXT;
BEGIN
    SELECT version, dirty INTO current_version, current_dirty
      FROM public.schema_migrations FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'schema_migrations has no readable version row; refusing recovery';
    END IF;
    IF current_dirty THEN
        RAISE EXCEPTION 'schema_migrations is dirty at version %; refusing automatic recovery', current_version;
    END IF;
    IF current_version NOT IN (4, 6) THEN
        RAISE EXCEPTION 'expected schema_migrations version 4 or 6, found %; refusing recovery', current_version;
    END IF;

    SELECT pg_get_constraintdef(oid) INTO overlap_definition
      FROM pg_constraint
     WHERE conrelid = 'public.academic_years'::regclass
       AND conname = 'academic_years_period_no_overlap' AND contype = 'x';
    IF overlap_definition IS NULL OR position('&&' IN overlap_definition) = 0 THEN
        RAISE EXCEPTION 'migration 000005 exclusion constraint is missing or unexpected';
    END IF;

    IF to_regclass('public.migration_000006_student_class_history_validity_backup') IS NULL THEN
        RAISE EXCEPTION '000006 backup table is missing; refusing metadata-only recovery';
    END IF;
    IF (
        SELECT count(*) FROM information_schema.columns
         WHERE table_schema = 'public'
           AND table_name = 'migration_000006_student_class_history_validity_backup'
    ) <> 5 OR (
        SELECT count(*)
          FROM information_schema.columns
         WHERE table_schema = 'public'
           AND table_name = 'migration_000006_student_class_history_validity_backup'
           AND (
             (column_name = 'student_class_history_id' AND data_type = 'uuid') OR
             (column_name = 'original_validity' AND udt_name = 'daterange') OR
             (column_name = 'normalized_validity' AND udt_name = 'daterange') OR
             (column_name = 'original_updated_at' AND data_type = 'timestamp with time zone') OR
             (column_name = 'normalized_updated_at' AND data_type = 'timestamp with time zone')
           )
    ) <> 5 THEN
        RAISE EXCEPTION '000006 backup table columns do not match expected contract';
    END IF;
    EXECUTE 'SELECT count(*) FROM public.migration_000006_student_class_history_validity_backup'
       INTO backup_rows;
    IF backup_rows <> 0 THEN
        RAISE EXCEPTION 'backup table contains % rows; refusing metadata-only recovery', backup_rows;
    END IF;

    WITH expected(table_name, trigger_name, function_name) AS (
        VALUES
          ('student_class_histories', 'student_class_histories_academic_year_bounds_trigger', 'enforce_student_class_history_academic_year_bounds'),
          ('academic_years', 'academic_years_history_bounds_trigger', 'enforce_academic_year_history_bounds'),
          ('academic_years', 'academic_years_close_requires_settled_accounts_trigger', 'enforce_academic_year_close_requires_settled_accounts'),
          ('savings_accounts', 'savings_accounts_creation_eligibility_trigger', 'enforce_savings_account_creation_eligibility'),
          ('savings_accounts', 'savings_accounts_no_reopen_after_year_close_trigger', 'enforce_savings_account_no_reopen_after_year_close'),
          ('class_transfer_requests', 'class_transfer_open_academic_year_trigger', 'enforce_class_transfer_open_academic_year'),
          ('transactions', 'transactions_cancel_requires_open_account_trigger', 'enforce_transaction_cancel_requires_open_account')
    )
    SELECT string_agg(e.table_name || '.' || e.trigger_name || ' -> ' || e.function_name, ', ')
      INTO missing_objects
      FROM expected e
     WHERE NOT EXISTS (
        SELECT 1
          FROM pg_trigger t
          JOIN pg_class c ON c.oid = t.tgrelid
          JOIN pg_namespace n ON n.oid = c.relnamespace
          JOIN pg_proc p ON p.oid = t.tgfoid
         WHERE n.nspname = 'public' AND c.relname = e.table_name
           AND t.tgname = e.trigger_name AND p.proname = e.function_name
           AND NOT t.tgisinternal
     );
    IF missing_objects IS NOT NULL THEN
        RAISE EXCEPTION 'missing/mismatched 000006 trigger wiring: %', missing_objects;
    END IF;

    IF to_regprocedure('public.enforce_student_class_history_academic_year_bounds()') IS NULL
       OR position('student class history period must be inside academic year period' IN pg_get_functiondef(to_regprocedure('public.enforce_student_class_history_academic_year_bounds()'))) = 0
       OR to_regprocedure('public.enforce_academic_year_history_bounds()') IS NULL
       OR position('academic year period cannot exclude existing student class history' IN pg_get_functiondef(to_regprocedure('public.enforce_academic_year_history_bounds()'))) = 0
       OR to_regprocedure('public.enforce_academic_year_close_requires_settled_accounts()') IS NULL
       OR position('academic year cannot close while savings accounts are open' IN pg_get_functiondef(to_regprocedure('public.enforce_academic_year_close_requires_settled_accounts()'))) = 0
       OR to_regprocedure('public.enforce_savings_account_creation_eligibility()') IS NULL
       OR position('savings account cannot be created for a LEFT student' IN pg_get_functiondef(to_regprocedure('public.enforce_savings_account_creation_eligibility()'))) = 0
       OR to_regprocedure('public.enforce_savings_account_no_reopen_after_year_close()') IS NULL
       OR position('savings account cannot be reopened after academic year close' IN pg_get_functiondef(to_regprocedure('public.enforce_savings_account_no_reopen_after_year_close()'))) = 0
       OR to_regprocedure('public.enforce_class_transfer_open_academic_year()') IS NULL
       OR position('class transfer requests require an OPEN academic year' IN pg_get_functiondef(to_regprocedure('public.enforce_class_transfer_open_academic_year()'))) = 0
       OR to_regprocedure('public.enforce_transaction_cancel_requires_open_account()') IS NULL
       OR position('transactions cannot be cancelled when the savings account is settled' IN pg_get_functiondef(to_regprocedure('public.enforce_transaction_cancel_requires_open_account()'))) = 0
    THEN
        RAISE EXCEPTION 'one or more 000006 function definitions are missing or unexpected';
    END IF;

    SELECT indexdef INTO index_definition FROM pg_indexes
     WHERE schemaname = 'public'
       AND indexname = 'class_transfer_requests_one_pending_per_student_year_idx';
    IF index_definition IS NULL
       OR position('(student_id, academic_year_id)' IN index_definition) = 0
       OR position('status = ''PENDING''' IN index_definition) = 0 THEN
        RAISE EXCEPTION '000006 pending-transfer unique index is missing or unexpected';
    END IF;

    SELECT pg_get_constraintdef(oid) INTO rejection_definition
      FROM pg_constraint
     WHERE conrelid = 'public.class_transfer_requests'::regclass
       AND conname = 'class_transfer_requests_rejection_reason_check'
       AND contype = 'c' AND convalidated;
    IF rejection_definition IS NULL
       OR position('REJECTED' IN rejection_definition) = 0
       OR position('btrim' IN rejection_definition) = 0 THEN
        RAISE EXCEPTION '000006 rejection-reason constraint is missing/unvalidated/unexpected';
    END IF;

    IF current_version = 4 THEN
        UPDATE public.schema_migrations SET version = 6, dirty = FALSE;
        RAISE NOTICE 'Validated 000005 and 000006 objects; schema_migrations reconciled to version 6';
    ELSE
        RAISE NOTICE 'schema_migrations already records version 6; validation passed';
    END IF;
END;
$recovery$;
COMMIT;

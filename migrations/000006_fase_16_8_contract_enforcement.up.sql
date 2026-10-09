BEGIN;

-- Normalize existing current histories to the end of their academic year.
-- Invalid finite histories are not silently rewritten; operators must resolve them first.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM student_class_histories h
          JOIN academic_years y ON y.id = h.academic_year_id
         WHERE lower_inf(h.validity)
            OR lower(h.validity) < y.start_date
            OR lower(h.validity) >= y.end_date
            OR (NOT upper_inf(h.validity) AND upper(h.validity) > y.end_date)
    ) THEN
        RAISE EXCEPTION 'cannot enforce student class history bounds: existing periods fall outside their academic year';
    END IF;
END;
$$;

CREATE TABLE migration_000006_student_class_history_validity_backup (
    student_class_history_id UUID PRIMARY KEY,
    original_validity DATERANGE NOT NULL,
    normalized_validity DATERANGE NOT NULL,
    original_updated_at TIMESTAMPTZ NOT NULL,
    normalized_updated_at TIMESTAMPTZ NOT NULL
);

INSERT INTO migration_000006_student_class_history_validity_backup (
    student_class_history_id,
    original_validity,
    normalized_validity,
    original_updated_at,
    normalized_updated_at
)
SELECT
    h.id,
    h.validity,
    daterange(lower(h.validity), y.end_date, '[)'),
    h.updated_at,
    NOW()
FROM student_class_histories h
JOIN academic_years y ON y.id = h.academic_year_id
WHERE upper_inf(h.validity);

UPDATE student_class_histories h
   SET validity = b.normalized_validity,
       updated_at = b.normalized_updated_at
  FROM migration_000006_student_class_history_validity_backup b
 WHERE h.id = b.student_class_history_id;

-- DEC-16-02: student class history must be contained in its academic year.
CREATE FUNCTION enforce_student_class_history_academic_year_bounds()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    year_start DATE;
    year_end DATE;
BEGIN
    SELECT start_date, end_date
      INTO year_start, year_end
      FROM academic_years
     WHERE id = NEW.academic_year_id
     FOR SHARE;

    IF NOT FOUND THEN
        RETURN NEW;
    END IF;

    IF lower_inf(NEW.validity)
       OR upper_inf(NEW.validity)
       OR lower(NEW.validity) < year_start
       OR lower(NEW.validity) >= year_end
       OR upper(NEW.validity) > year_end THEN
        RAISE EXCEPTION 'student class history period must be inside academic year period'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'student_class_histories_academic_year_bounds_check';
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER student_class_histories_academic_year_bounds_trigger
BEFORE INSERT OR UPDATE OF academic_year_id, validity
ON student_class_histories
FOR EACH ROW
EXECUTE FUNCTION enforce_student_class_history_academic_year_bounds();

CREATE FUNCTION enforce_academic_year_history_bounds()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM student_class_histories h
         WHERE h.academic_year_id = NEW.id
           AND (
               lower_inf(h.validity)
               OR upper_inf(h.validity)
               OR lower(h.validity) < NEW.start_date
               OR lower(h.validity) >= NEW.end_date
               OR upper(h.validity) > NEW.end_date
           )
    ) THEN
        RAISE EXCEPTION 'academic year period cannot exclude existing student class history'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'academic_years_history_bounds_check';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER academic_years_history_bounds_trigger
BEFORE UPDATE OF start_date, end_date
ON academic_years
FOR EACH ROW
EXECUTE FUNCTION enforce_academic_year_history_bounds();

-- DEC-16-03: all accounts must be settled before an academic year closes.
CREATE FUNCTION enforce_academic_year_close_requires_settled_accounts()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.status = 'CLOSED'
       AND OLD.status IS DISTINCT FROM 'CLOSED'
       AND EXISTS (
           SELECT 1
             FROM savings_accounts
            WHERE academic_year_id = NEW.id
              AND status <> 'SETTLED'
       ) THEN
        RAISE EXCEPTION 'academic year cannot close while savings accounts are open'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'academic_years_close_requires_settled_accounts';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER academic_years_close_requires_settled_accounts_trigger
BEFORE UPDATE OF status
ON academic_years
FOR EACH ROW
EXECUTE FUNCTION enforce_academic_year_close_requires_settled_accounts();

-- DEC-16-05/06: new accounts require an ACTIVE student and OPEN academic year.
CREATE FUNCTION enforce_savings_account_creation_eligibility()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    student_status TEXT;
    year_status TEXT;
BEGIN
    SELECT status INTO year_status
      FROM academic_years
     WHERE id = NEW.academic_year_id
     FOR SHARE;

    SELECT status INTO student_status
      FROM students
     WHERE id = NEW.student_id
     FOR SHARE;

    IF year_status IS DISTINCT FROM 'OPEN' THEN
        RAISE EXCEPTION 'savings account requires an OPEN academic year'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'savings_accounts_academic_year_must_be_open';
    END IF;

    IF student_status IS DISTINCT FROM 'ACTIVE' THEN
        RAISE EXCEPTION 'savings account cannot be created for a LEFT student'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'savings_accounts_student_must_be_active';
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER savings_accounts_creation_eligibility_trigger
BEFORE INSERT
ON savings_accounts
FOR EACH ROW
EXECUTE FUNCTION enforce_savings_account_creation_eligibility();

-- DEC-16-09: a settled account cannot be reopened while its academic year is CLOSED.
CREATE FUNCTION enforce_savings_account_no_reopen_after_year_close()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    year_status TEXT;
BEGIN
    IF NEW.status = 'OPEN' AND OLD.status IS DISTINCT FROM 'OPEN' THEN
        SELECT status INTO year_status
          FROM academic_years
         WHERE id = OLD.academic_year_id
         FOR SHARE;

        IF year_status = 'CLOSED' THEN
            RAISE EXCEPTION 'savings account cannot be reopened after academic year close'
                USING ERRCODE = '23514',
                      CONSTRAINT = 'savings_accounts_no_reopen_after_year_close';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER savings_accounts_no_reopen_after_year_close_trigger
BEFORE UPDATE OF status
ON savings_accounts
FOR EACH ROW
EXECUTE FUNCTION enforce_savings_account_no_reopen_after_year_close();

-- DEC-16-07/08: transfers can only be created/processed in OPEN years,
-- with at most one PENDING request per student and academic year.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM class_transfer_requests
         WHERE status = 'PENDING'
         GROUP BY student_id, academic_year_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'cannot enforce one pending transfer per student and academic year: duplicate pending requests exist';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM class_transfer_requests
         WHERE (status = 'REJECTED' AND (rejection_reason IS NULL OR btrim(rejection_reason) = ''))
            OR (status <> 'REJECTED' AND rejection_reason IS NOT NULL)
    ) THEN
        RAISE EXCEPTION 'cannot enforce transfer rejection reason contract: existing request data is inconsistent';
    END IF;
END;
$$;

CREATE UNIQUE INDEX class_transfer_requests_one_pending_per_student_year_idx
    ON class_transfer_requests (student_id, academic_year_id)
    WHERE status = 'PENDING';

CREATE FUNCTION enforce_class_transfer_open_academic_year()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    year_status TEXT;
BEGIN
    SELECT status INTO year_status
      FROM academic_years
     WHERE id = NEW.academic_year_id
     FOR SHARE;

    IF year_status IS DISTINCT FROM 'OPEN' THEN
        RAISE EXCEPTION 'class transfer requests require an OPEN academic year'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'class_transfer_requests_academic_year_must_be_open';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER class_transfer_open_academic_year_trigger
BEFORE INSERT OR UPDATE OF academic_year_id, status
ON class_transfer_requests
FOR EACH ROW
EXECUTE FUNCTION enforce_class_transfer_open_academic_year();

-- FIN-CANCEL-02/FIN-SETTLEMENT-05: transactions of SETTLED accounts cannot be cancelled.
CREATE FUNCTION enforce_transaction_cancel_requires_open_account()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    account_status TEXT;
BEGIN
    IF NEW.status = 'CANCELLED' AND OLD.status IS DISTINCT FROM 'CANCELLED' THEN
        SELECT status INTO account_status
          FROM savings_accounts
         WHERE id = NEW.savings_account_id
         FOR SHARE;

        IF account_status IS DISTINCT FROM 'OPEN' THEN
            RAISE EXCEPTION 'transactions cannot be cancelled when the savings account is settled'
                USING ERRCODE = '23514',
                      CONSTRAINT = 'transactions_cancel_requires_open_account';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER transactions_cancel_requires_open_account_trigger
BEFORE UPDATE OF status
ON transactions
FOR EACH ROW
EXECUTE FUNCTION enforce_transaction_cancel_requires_open_account();

-- DEC-16-13: rejection reason is present only for REJECTED requests.
ALTER TABLE class_transfer_requests
    ADD CONSTRAINT class_transfer_requests_rejection_reason_check
    CHECK (
        (status = 'REJECTED' AND rejection_reason IS NOT NULL AND btrim(rejection_reason) <> '')
        OR
        (status <> 'REJECTED' AND rejection_reason IS NULL)
    );

COMMIT;

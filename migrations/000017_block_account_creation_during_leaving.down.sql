-- Lock yang sama mencegah INSERT melihat definisi trigger setengah rollback.
LOCK TABLE savings_accounts IN SHARE ROW EXCLUSIVE MODE;

CREATE OR REPLACE FUNCTION enforce_savings_account_creation_eligibility()
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

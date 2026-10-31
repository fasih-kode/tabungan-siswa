-- Cegah INSERT rekening saat transaksi Start menambahkan proses pindah.
-- Lock tabel menunggu INSERT lama selesai sebelum trigger diganti dan menahan
-- INSERT baru sampai fungsi trigger versi baru telah aktif.
LOCK TABLE savings_accounts IN SHARE ROW EXCLUSIVE MODE;

CREATE OR REPLACE FUNCTION enforce_savings_account_creation_eligibility()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    year_status TEXT;
    student_status TEXT;
BEGIN
    SELECT status
      INTO year_status
      FROM academic_years
     WHERE id = NEW.academic_year_id
     FOR SHARE;

    IF year_status IS DISTINCT FROM 'OPEN' THEN
        RAISE EXCEPTION 'savings account requires an OPEN academic year'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'savings_accounts_academic_year_must_be_open';
    END IF;

    SELECT status
      INTO student_status
      FROM students
     WHERE id = NEW.student_id
     FOR SHARE;

    IF student_status IS DISTINCT FROM 'ACTIVE' THEN
        RAISE EXCEPTION 'savings account cannot be created for a LEFT student'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'savings_accounts_student_must_be_active';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM student_leaving_requests
         WHERE student_id = NEW.student_id
           AND status = 'IN_PROGRESS'
    ) THEN
        RAISE EXCEPTION 'savings account cannot be created during an active student leaving process'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'savings_accounts_student_leaving_in_progress';
    END IF;

    RETURN NEW;
END;
$$;

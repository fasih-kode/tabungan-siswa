BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM students WHERE status = 'GRADUATED') THEN
        RAISE EXCEPTION 'cannot roll back student graduation while graduated students exist';
    END IF;
END;
$$;

ALTER TABLE students
    DROP CONSTRAINT students_status_check;

ALTER TABLE students
    ADD CONSTRAINT students_status_check
    CHECK (status IN ('ACTIVE', 'LEFT'));

COMMIT;

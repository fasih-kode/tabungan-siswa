BEGIN;

ALTER TABLE students
    DROP CONSTRAINT students_status_check;

ALTER TABLE students
    ADD CONSTRAINT students_status_check
    CHECK (status IN ('ACTIVE', 'LEFT', 'GRADUATED'));

COMMIT;

DO $migration$
BEGIN
    LOCK TABLE student_leaving_requests IN ACCESS EXCLUSIVE MODE;

    IF EXISTS (SELECT 1 FROM student_leaving_requests) THEN
        RAISE EXCEPTION 'cannot drop student_leaving_requests because it contains process history; preserve the data before rolling back'
            USING ERRCODE = '55000';
    END IF;

    DROP TABLE student_leaving_requests;
END
$migration$;

DO $migration$
BEGIN
    LOCK TABLE leaving_transaction_requests, savings_settlements
        IN ACCESS EXCLUSIVE MODE;

    IF EXISTS (
        SELECT 1
        FROM leaving_transaction_requests
        WHERE student_leaving_request_id IS NOT NULL
    ) OR EXISTS (
        SELECT 1
        FROM savings_settlements
        WHERE student_leaving_request_id IS NOT NULL
    ) THEN
        RAISE EXCEPTION 'cannot drop student leaving financial links because they contain references; preserve the links before rolling back'
            USING ERRCODE = '55000';
    END IF;

    ALTER TABLE leaving_transaction_requests
        DROP COLUMN student_leaving_request_id;

    ALTER TABLE savings_settlements
        DROP COLUMN student_leaving_request_id;
END
$migration$;

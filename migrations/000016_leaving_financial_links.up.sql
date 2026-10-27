ALTER TABLE savings_settlements
    ADD COLUMN student_leaving_request_id UUID NULL,
    ADD CONSTRAINT savings_settlements_student_leaving_request_id_fkey
        FOREIGN KEY (student_leaving_request_id)
        REFERENCES student_leaving_requests (id);

ALTER TABLE leaving_transaction_requests
    ADD COLUMN student_leaving_request_id UUID NULL,
    ADD CONSTRAINT leaving_transaction_requests_student_leaving_request_id_fkey
        FOREIGN KEY (student_leaving_request_id)
        REFERENCES student_leaving_requests (id);

CREATE INDEX savings_settlements_student_leaving_request_id_idx
    ON savings_settlements (student_leaving_request_id);

CREATE INDEX leaving_transaction_requests_student_leaving_request_id_idx
    ON leaving_transaction_requests (student_leaving_request_id);

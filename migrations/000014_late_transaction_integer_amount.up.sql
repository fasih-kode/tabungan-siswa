ALTER TABLE leaving_transaction_requests
    ADD CONSTRAINT leaving_transaction_requests_amount_integer_check
    CHECK (amount = trunc(amount, 0));

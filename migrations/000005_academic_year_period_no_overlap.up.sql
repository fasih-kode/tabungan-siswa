ALTER TABLE academic_years
    ADD CONSTRAINT academic_years_period_no_overlap
    EXCLUDE USING gist (
        (daterange(start_date, end_date, '[)')) WITH &&
    );

CREATE FUNCTION enforce_academic_year_create_after_previous_closed()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM academic_years
         WHERE status <> 'CLOSED'
    ) THEN
        RAISE EXCEPTION 'new academic year requires all previous years to be CLOSED'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'academic_years_create_requires_closed_previous';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER academic_years_create_requires_closed_previous_trigger
BEFORE INSERT ON academic_years
FOR EACH ROW
EXECUTE FUNCTION enforce_academic_year_create_after_previous_closed();

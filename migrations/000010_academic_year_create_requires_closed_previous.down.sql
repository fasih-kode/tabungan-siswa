DROP TRIGGER IF EXISTS academic_years_create_requires_closed_previous_trigger
    ON academic_years;
DROP FUNCTION IF EXISTS enforce_academic_year_create_after_previous_closed();

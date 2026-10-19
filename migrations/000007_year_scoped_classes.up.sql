BEGIN;

ALTER TABLE classes DROP CONSTRAINT classes_name_level_unique;
ALTER TABLE classes
    ADD COLUMN academic_year_id UUID REFERENCES academic_years(id),
    ADD COLUMN effective_date DATE,
    ADD CONSTRAINT classes_year_configuration_check CHECK (
        (academic_year_id IS NULL AND effective_date IS NULL) OR
        (academic_year_id IS NOT NULL AND effective_date IS NOT NULL)
    ),
    ADD CONSTRAINT classes_year_name_level_unique UNIQUE (academic_year_id, name, level);

CREATE UNIQUE INDEX classes_legacy_name_level_unique ON classes (name, level) WHERE academic_year_id IS NULL;

-- Global classes were visible in every year. Preserve that catalogue for
-- existing years, and preserve each year-dependent reference when splitting it.
CREATE TEMP TABLE class_year_mapping ON COMMIT DROP AS
SELECT c.id AS original_id, y.id AS year_id,
       CASE WHEN row_number() OVER (PARTITION BY c.id ORDER BY y.start_date, y.id) = 1
            THEN c.id ELSE gen_random_uuid() END AS class_id
FROM classes c CROSS JOIN academic_years y;

INSERT INTO classes (id, name, level, academic_year_id, effective_date, created_at, updated_at)
SELECT m.class_id, c.name, c.level, m.year_id, y.start_date, c.created_at, c.updated_at
FROM class_year_mapping m
JOIN classes c ON c.id = m.original_id
JOIN academic_years y ON y.id = m.year_id
WHERE m.class_id <> m.original_id;

UPDATE classes c SET academic_year_id = m.year_id, effective_date = y.start_date
FROM class_year_mapping m JOIN academic_years y ON y.id = m.year_id
WHERE c.id = m.original_id AND m.class_id = m.original_id;

UPDATE student_class_histories h SET class_id = m.class_id
FROM class_year_mapping m WHERE h.class_id = m.original_id AND h.academic_year_id = m.year_id;
UPDATE teacher_class_assignments a SET class_id = m.class_id
FROM class_year_mapping m WHERE a.class_id = m.original_id AND a.academic_year_id = m.year_id;
UPDATE class_transfer_requests r SET from_class_id = m.class_id
FROM class_year_mapping m WHERE r.from_class_id = m.original_id AND r.academic_year_id = m.year_id;
UPDATE class_transfer_requests r SET to_class_id = m.class_id
FROM class_year_mapping m WHERE r.to_class_id = m.original_id AND r.academic_year_id = m.year_id;

CREATE FUNCTION enforce_class_effective_date_bounds() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE year_start DATE; year_end DATE;
BEGIN
    IF NEW.academic_year_id IS NULL THEN RETURN NEW; END IF;
    SELECT start_date, end_date INTO year_start, year_end FROM academic_years
      WHERE id = NEW.academic_year_id FOR SHARE;
    IF NEW.effective_date < year_start OR NEW.effective_date >= year_end THEN
        RAISE EXCEPTION 'class effective date must be inside academic year'
          USING ERRCODE = '23514', CONSTRAINT = 'classes_effective_date_bounds_check';
    END IF;
    IF TG_OP = 'UPDATE' AND OLD.academic_year_id IS DISTINCT FROM NEW.academic_year_id AND (
        EXISTS (SELECT 1 FROM student_class_histories WHERE class_id = OLD.id) OR
        EXISTS (SELECT 1 FROM teacher_class_assignments WHERE class_id = OLD.id) OR
        EXISTS (SELECT 1 FROM class_transfer_requests WHERE from_class_id = OLD.id OR to_class_id = OLD.id)
    ) THEN
        RAISE EXCEPTION 'cannot change academic year of a referenced class'
          USING ERRCODE = '23514', CONSTRAINT = 'classes_referenced_year_check';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER classes_effective_date_bounds_trigger
BEFORE INSERT OR UPDATE OF academic_year_id, effective_date ON classes
FOR EACH ROW EXECUTE FUNCTION enforce_class_effective_date_bounds();

CREATE FUNCTION enforce_academic_year_class_bounds() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM classes WHERE academic_year_id = NEW.id AND
        (effective_date < NEW.start_date OR effective_date >= NEW.end_date)) THEN
        RAISE EXCEPTION 'academic year cannot exclude configured class effective dates'
          USING ERRCODE = '23514', CONSTRAINT = 'academic_years_class_bounds_check';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER academic_years_class_bounds_trigger BEFORE UPDATE OF start_date, end_date ON academic_years
FOR EACH ROW EXECUTE FUNCTION enforce_academic_year_class_bounds();

-- Legacy unconfigured rows (possible only when no academic year existed) are
-- retained. All configured references must match their academic year.
CREATE FUNCTION enforce_class_reference_year() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE field TEXT; class_year UUID;
BEGIN
    FOREACH field IN ARRAY TG_ARGV LOOP
        SELECT academic_year_id INTO class_year FROM classes
          WHERE id = (to_jsonb(NEW)->>field)::uuid;
        IF class_year IS NOT NULL AND class_year <> NEW.academic_year_id THEN
            RAISE EXCEPTION 'class does not belong to the academic year'
              USING ERRCODE = '23514', CONSTRAINT = 'class_reference_academic_year_check';
        END IF;
    END LOOP;
    RETURN NEW;
END;
$$;
CREATE TRIGGER student_class_history_year_trigger BEFORE INSERT OR UPDATE OF class_id, academic_year_id ON student_class_histories
FOR EACH ROW EXECUTE FUNCTION enforce_class_reference_year('class_id');
CREATE TRIGGER teacher_class_assignment_year_trigger BEFORE INSERT OR UPDATE OF class_id, academic_year_id ON teacher_class_assignments
FOR EACH ROW EXECUTE FUNCTION enforce_class_reference_year('class_id');
CREATE TRIGGER class_transfer_reference_year_trigger BEFORE INSERT OR UPDATE OF from_class_id, to_class_id, academic_year_id ON class_transfer_requests
FOR EACH ROW EXECUTE FUNCTION enforce_class_reference_year('from_class_id','to_class_id');

COMMIT;

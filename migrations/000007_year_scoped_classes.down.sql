BEGIN;
DROP TRIGGER class_transfer_reference_year_trigger ON class_transfer_requests;
DROP TRIGGER teacher_class_assignment_year_trigger ON teacher_class_assignments;
DROP TRIGGER student_class_history_year_trigger ON student_class_histories;
DROP FUNCTION enforce_class_reference_year();
DROP INDEX classes_legacy_name_level_unique;
DROP TRIGGER academic_years_class_bounds_trigger ON academic_years;
DROP FUNCTION enforce_academic_year_class_bounds();
DROP TRIGGER classes_effective_date_bounds_trigger ON classes;
DROP FUNCTION enforce_class_effective_date_bounds();
-- Restore global class identities by merging equal name/level pairs.
CREATE TEMP TABLE class_merge_mapping ON COMMIT DROP AS
SELECT id, first_value(id) OVER (PARTITION BY name, level ORDER BY created_at, id) AS target_id FROM classes;
UPDATE student_class_histories h SET class_id = m.target_id FROM class_merge_mapping m WHERE h.class_id = m.id;
UPDATE teacher_class_assignments a SET class_id = m.target_id FROM class_merge_mapping m WHERE a.class_id = m.id;
UPDATE class_transfer_requests r SET from_class_id = m.target_id FROM class_merge_mapping m WHERE r.from_class_id = m.id;
UPDATE class_transfer_requests r SET to_class_id = m.target_id FROM class_merge_mapping m WHERE r.to_class_id = m.id;
DELETE FROM classes c USING class_merge_mapping m WHERE c.id = m.id AND m.id <> m.target_id;
ALTER TABLE classes DROP COLUMN academic_year_id, DROP COLUMN effective_date;
ALTER TABLE classes ADD CONSTRAINT classes_name_level_unique UNIQUE (name, level);
COMMIT;

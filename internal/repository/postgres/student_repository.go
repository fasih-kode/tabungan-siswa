package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

var _ repository.StudentRepository = (*StudentRepository)(nil)

type StudentRepository struct {
	db database.DBTX
}

func NewStudentRepository(db database.DBTX) *StudentRepository {
	return &StudentRepository{db: db}
}

func (r *StudentRepository) Create(
	ctx context.Context,
	student domain.Student,
) error {
	const query = `
		INSERT INTO students (
			id,
			user_id,
			nis,
			nisn,
			name,
			status,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		student.ID,
		student.UserID,
		student.NIS,
		student.NISN,
		student.Name,
		student.Status,
		student.CreatedAt,
		student.UpdatedAt,
	)
	if err != nil {
		return translateError("create student", err)
	}

	return nil
}

func (r *StudentRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.Student, error) {
	const query = `
		SELECT
			id,
			user_id,
			nis,
			nisn,
			name,
			status,
			created_at,
			updated_at
		FROM students
		WHERE id = $1
	`

	return r.getOne(ctx, query, id)
}

func (r *StudentRepository) GetByIDForUpdate(
	ctx context.Context,
	id uuid.UUID,
) (domain.Student, error) {
	const query = `
		SELECT
			id,
			user_id,
			nis,
			nisn,
			name,
			status,
			created_at,
			updated_at
		FROM students
		WHERE id = $1
		FOR UPDATE
	`
	return r.getOne(ctx, query, id)
}

func (r *StudentRepository) GetByNIS(
	ctx context.Context,
	nis string,
) (domain.Student, error) {
	const query = `
		SELECT
			id,
			user_id,
			nis,
			nisn,
			name,
			status,
			created_at,
			updated_at
		FROM students
		WHERE nis = $1
	`

	return r.getOne(ctx, query, nis)
}

func (r *StudentRepository) GetByNISN(
	ctx context.Context,
	nisn string,
) (domain.Student, error) {
	const query = `
		SELECT
			id,
			user_id,
			nis,
			nisn,
			name,
			status,
			created_at,
			updated_at
		FROM students
		WHERE nisn = $1
	`

	return r.getOne(ctx, query, nisn)
}

func (r *StudentRepository) GetByUserID(
	ctx context.Context,
	userID uuid.UUID,
) (domain.Student, error) {
	const query = `
		SELECT
			id,
			user_id,
			nis,
			nisn,
			name,
			status,
			created_at,
			updated_at
		FROM students
		WHERE user_id = $1
	`

	return r.getOne(ctx, query, userID)
}

func (r *StudentRepository) List(
	ctx context.Context,
	options repository.ListOptions,
) ([]domain.Student, error) {
	const query = `
		SELECT
			id,
			user_id,
			nis,
			nisn,
			name,
			status,
			created_at,
			updated_at
		FROM students
		WHERE (
			$1 = '' OR
			name ILIKE '%' || $1 || '%' OR
			COALESCE(nis, '') ILIKE '%' || $1 || '%' OR
			COALESCE(nisn, '') ILIKE '%' || $1 || '%'
		)
		AND ($2 = '' OR status = $2)
		AND (
			NOT $3 OR NOT EXISTS (
				SELECT 1 FROM student_class_histories h
				WHERE h.student_id = students.id
			)
		)
		ORDER BY name ASC, id ASC
		LIMIT $4 OFFSET $5
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		options.Search,
		options.StudentStatus,
		options.UnassignedOnly,
		options.Limit,
		options.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list students: %w", err)
	}
	defer rows.Close()

	students := make([]domain.Student, 0)

	for rows.Next() {
		student, err := scanStudent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan student: %w", err)
		}

		students = append(students, student)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate students: %w", err)
	}

	return students, nil
}

func (r *StudentRepository) ListByClassAndAcademicYear(
	ctx context.Context,
	classID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.Student, error) {
	const query = `
		SELECT DISTINCT
			s.id,
			s.user_id,
			s.nis,
			s.nisn,
			s.name,
			s.status,
			s.created_at,
			s.updated_at
		FROM students s
		INNER JOIN student_class_histories h
			ON h.student_id = s.id
			AND h.class_id = $1
			AND h.academic_year_id = $2
		WHERE NOT isempty(h.validity)
			AND (
				$5 = '' OR
				s.name ILIKE '%' || $5 || '%' OR
				COALESCE(s.nis, '') ILIKE '%' || $5 || '%' OR
				COALESCE(s.nisn, '') ILIKE '%' || $5 || '%'
			)
			AND ($6 = '' OR s.status = $6)
		ORDER BY s.name ASC, s.id ASC
		LIMIT $3
		OFFSET $4
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		classID,
		academicYearID,
		options.Limit,
		options.Offset,
		options.Search,
		options.StudentStatus,
	)
	if err != nil {
		return nil, fmt.Errorf("list students by class and academic year: %w", err)
	}
	defer rows.Close()

	students := make([]domain.Student, 0)

	for rows.Next() {
		student, err := scanStudent(rows)
		if err != nil {
			return nil, fmt.Errorf(
				"scan student by class and academic year: %w",
				err,
			)
		}

		students = append(students, student)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate students by class and academic year: %w",
			err,
		)
	}

	return students, nil
}

func (r *StudentRepository) Update(
	ctx context.Context,
	student domain.Student,
) error {
	const query = `
		UPDATE students
		SET
			user_id = $1,
			nis = $2,
			nisn = $3,
			name = $4,
			status = $5,
			updated_at = $6
		WHERE id = $7
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		student.UserID,
		student.NIS,
		student.NISN,
		student.Name,
		student.Status,
		student.UpdatedAt,
		student.ID,
	)
	if err != nil {
		return translateError("update student", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update student rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r *StudentRepository) getOne(
	ctx context.Context,
	query string,
	arg any,
) (domain.Student, error) {
	var student domain.Student

	err := r.db.QueryRowContext(
		ctx,
		query,
		arg,
	).Scan(
		&student.ID,
		&student.UserID,
		&student.NIS,
		&student.NISN,
		&student.Name,
		&student.Status,
		&student.CreatedAt,
		&student.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Student{}, repository.ErrNotFound
		}

		return domain.Student{}, fmt.Errorf("get student: %w", err)
	}

	return student, nil
}

func scanStudent(rows *sql.Rows) (domain.Student, error) {
	var student domain.Student

	err := rows.Scan(
		&student.ID,
		&student.UserID,
		&student.NIS,
		&student.NISN,
		&student.Name,
		&student.Status,
		&student.CreatedAt,
		&student.UpdatedAt,
	)
	if err != nil {
		return domain.Student{}, err
	}

	return student, nil
}

// ListDirectory selects one placement per student: the latest history in the
// selected year, or the latest year/history when no year filter is supplied.
func (r *StudentRepository) ListDirectory(ctx context.Context, opts repository.ListOptions) ([]repository.StudentDirectoryEntry, error) {
	const query = `SELECT s.id, s.user_id, s.nis, s.nisn, s.name, s.status, s.created_at, s.updated_at,
  p.academic_year_id, p.class_id, COALESCE(p.year_name,''), COALESCE(p.class_name,'')` + studentDirectoryFromWhere + `
 ORDER BY s.name, s.id LIMIT $7 OFFSET $8`
	rows, err := r.db.QueryContext(ctx, query, opts.AcademicYearID, opts.ClassID, opts.Search, opts.StudentStatus, opts.UnassignedOnly, opts.TeacherID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, fmt.Errorf("list student directory: %w", err)
	}
	defer rows.Close()
	result := make([]repository.StudentDirectoryEntry, 0)
	for rows.Next() {
		var entry repository.StudentDirectoryEntry
		var year, class uuid.NullUUID
		s := &entry.Student
		if err := rows.Scan(&s.ID, &s.UserID, &s.NIS, &s.NISN, &s.Name, &s.Status, &s.CreatedAt, &s.UpdatedAt, &year, &class, &entry.AcademicYear, &entry.Class); err != nil {
			return nil, fmt.Errorf("scan student directory: %w", err)
		}
		if year.Valid {
			entry.AcademicYearID = year.UUID
		}
		if class.Valid {
			entry.ClassID = class.UUID
		}
		result = append(result, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate student directory: %w", err)
	}
	return result, nil
}

const studentDirectoryFromWhere = `
 FROM students s
 LEFT JOIN LATERAL (
  SELECT h.academic_year_id, h.class_id, y.name AS year_name, c.name AS class_name
  FROM student_class_histories h JOIN academic_years y ON y.id=h.academic_year_id JOIN classes c ON c.id=h.class_id
  WHERE h.student_id=s.id AND ($1::uuid='00000000-0000-0000-0000-000000000000' OR h.academic_year_id=$1)
  ORDER BY y.start_date DESC, lower(h.validity) DESC, h.id DESC LIMIT 1
 ) p ON TRUE
 WHERE ($2::uuid='00000000-0000-0000-0000-000000000000' OR p.class_id=$2)
 AND ($1::uuid='00000000-0000-0000-0000-000000000000' OR p.academic_year_id IS NOT NULL OR $5)
 AND (NOT $5 OR p.class_id IS NULL)
 AND ($6::uuid='00000000-0000-0000-0000-000000000000' OR EXISTS (
  SELECT 1 FROM teacher_class_assignments a
  WHERE a.user_id=$6 AND a.academic_year_id=p.academic_year_id AND a.class_id=p.class_id
 ))
 AND ($3='' OR s.name ILIKE '%'||$3||'%' OR s.nis ILIKE '%'||$3||'%' OR s.nisn ILIKE '%'||$3||'%')
 AND ($4='' OR s.status=$4)`

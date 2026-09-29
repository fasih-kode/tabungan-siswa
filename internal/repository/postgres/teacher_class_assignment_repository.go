package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ repository.TeacherClassAssignmentRepository = (*TeacherClassAssignmentRepository)(nil)

type TeacherClassAssignmentRepository struct {
	db database.DBTX
}

func NewTeacherClassAssignmentRepository(db database.DBTX) *TeacherClassAssignmentRepository {
	return &TeacherClassAssignmentRepository{db: db}
}

func (r *TeacherClassAssignmentRepository) Create(
	ctx context.Context,
	assignment domain.TeacherClassAssignment,
) error {
	const query = `
		INSERT INTO teacher_class_assignments (
			id,
			user_id,
			academic_year_id,
			class_id,
			validity,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			daterange($5::date, $6::date, '[)'),
			$7,
			$8
		)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		assignment.ID,
		assignment.UserID,
		assignment.AcademicYearID,
		assignment.ClassID,
		assignment.Period.From,
		assignment.Period.To,
		assignment.CreatedAt,
		assignment.UpdatedAt,
	)
	if err != nil {
		return translateTeacherClassAssignmentError(
			"create teacher class assignment",
			err,
		)
	}

	return nil
}

func (r *TeacherClassAssignmentRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.TeacherClassAssignment, error) {
	const query = `
		SELECT
			id,
			user_id,
			academic_year_id,
			class_id,
			lower(validity),
			upper(validity),
			created_at,
			updated_at
		FROM teacher_class_assignments
		WHERE id = $1
	`

	return r.getOne(ctx, query, id)
}

func (r *TeacherClassAssignmentRepository) ListByUserAndAcademicYear(
	ctx context.Context,
	userID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.TeacherClassAssignment, error) {
	const query = `
		SELECT
			id,
			user_id,
			academic_year_id,
			class_id,
			lower(validity),
			upper(validity),
			created_at,
			updated_at
		FROM teacher_class_assignments
		WHERE user_id = $1
			AND academic_year_id = $2
		ORDER BY lower(validity) ASC, id ASC
		LIMIT $3 OFFSET $4
	`

	return r.list(
		ctx,
		query,
		"list teacher class assignments by user and academic year",
		userID,
		academicYearID,
		options.Limit,
		options.Offset,
	)
}

func (r *TeacherClassAssignmentRepository) ListByClassAndAcademicYear(
	ctx context.Context,
	classID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.TeacherClassAssignment, error) {
	const query = `
		SELECT
			id,
			user_id,
			academic_year_id,
			class_id,
			lower(validity),
			upper(validity),
			created_at,
			updated_at
		FROM teacher_class_assignments
		WHERE class_id = $1
			AND academic_year_id = $2
		ORDER BY lower(validity) ASC, id ASC
		LIMIT $3 OFFSET $4
	`

	return r.list(
		ctx,
		query,
		"list teacher class assignments by class and academic year",
		classID,
		academicYearID,
		options.Limit,
		options.Offset,
	)
}

func (r *TeacherClassAssignmentRepository) GetCurrentByClassAndAcademicYear(
	ctx context.Context,
	classID uuid.UUID,
	academicYearID uuid.UUID,
	dateValue time.Time,
) (domain.TeacherClassAssignment, error) {
	const query = `
		SELECT
			id,
			user_id,
			academic_year_id,
			class_id,
			lower(validity),
			upper(validity),
			created_at,
			updated_at
		FROM teacher_class_assignments
		WHERE class_id = $1
			AND academic_year_id = $2
			AND validity @> $3::date
		ORDER BY lower(validity) DESC, id DESC
		LIMIT 1
	`

	return r.getOne(
		ctx,
		query,
		classID,
		academicYearID,
		dateValue,
	)
}

func (r *TeacherClassAssignmentRepository) getOne(
	ctx context.Context,
	query string,
	args ...any,
) (domain.TeacherClassAssignment, error) {
	var assignment domain.TeacherClassAssignment
	var upper sql.NullTime

	err := r.db.QueryRowContext(
		ctx,
		query,
		args...,
	).Scan(
		&assignment.ID,
		&assignment.UserID,
		&assignment.AcademicYearID,
		&assignment.ClassID,
		&assignment.Period.From,
		&upper,
		&assignment.CreatedAt,
		&assignment.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.TeacherClassAssignment{}, repository.ErrNotFound
		}

		return domain.TeacherClassAssignment{}, fmt.Errorf(
			"get teacher class assignment: %w",
			err,
		)
	}

	if upper.Valid {
		assignment.Period.To = &upper.Time
	}

	return assignment, nil
}

func (r *TeacherClassAssignmentRepository) list(
	ctx context.Context,
	query string,
	operation string,
	args ...any,
) ([]domain.TeacherClassAssignment, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}
	defer rows.Close()

	assignments := make([]domain.TeacherClassAssignment, 0)

	for rows.Next() {
		assignment, err := scanTeacherClassAssignment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan teacher class assignment: %w", err)
		}

		assignments = append(assignments, assignment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate teacher class assignments: %w", err)
	}

	return assignments, nil
}

func scanTeacherClassAssignment(
	rows *sql.Rows,
) (domain.TeacherClassAssignment, error) {
	var assignment domain.TeacherClassAssignment
	var upper sql.NullTime

	err := rows.Scan(
		&assignment.ID,
		&assignment.UserID,
		&assignment.AcademicYearID,
		&assignment.ClassID,
		&assignment.Period.From,
		&upper,
		&assignment.CreatedAt,
		&assignment.UpdatedAt,
	)
	if err != nil {
		return domain.TeacherClassAssignment{}, err
	}

	if upper.Valid {
		assignment.Period.To = &upper.Time
	}

	return assignment, nil
}

func translateTeacherClassAssignmentError(
	operation string,
	err error,
) error {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "23P01":
			return fmt.Errorf("%s: %w", operation, repository.ErrConflict)
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}

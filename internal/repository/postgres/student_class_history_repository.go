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

var _ repository.StudentClassHistoryRepository = (*StudentClassHistoryRepository)(nil)

type StudentClassHistoryRepository struct {
	db database.DBTX
}

func NewStudentClassHistoryRepository(db database.DBTX) *StudentClassHistoryRepository {
	return &StudentClassHistoryRepository{db: db}
}

func (r *StudentClassHistoryRepository) Create(
	ctx context.Context,
	history domain.StudentClassHistory,
) error {
	const query = `
		INSERT INTO student_class_histories (
			id,
			student_id,
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
		history.ID,
		history.StudentID,
		history.AcademicYearID,
		history.ClassID,
		history.Period.From,
		history.Period.To,
		history.CreatedAt,
		history.UpdatedAt,
	)
	if err != nil {
		return translateStudentClassHistoryError("create student class history", err)
	}

	return nil
}

func (r *StudentClassHistoryRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.StudentClassHistory, error) {
	const query = `
		SELECT
			id,
			student_id,
			academic_year_id,
			class_id,
			lower(validity),
			upper(validity),
			created_at,
			updated_at
		FROM student_class_histories
		WHERE id = $1
	`

	return r.getOne(ctx, query, id)
}

func (r *StudentClassHistoryRepository) ListByStudentAndAcademicYear(
	ctx context.Context,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.StudentClassHistory, error) {
	const query = `
		SELECT
			id,
			student_id,
			academic_year_id,
			class_id,
			lower(validity),
			upper(validity),
			created_at,
			updated_at
		FROM student_class_histories
		WHERE student_id = $1
			AND academic_year_id = $2
		ORDER BY lower(validity) ASC, id ASC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		studentID,
		academicYearID,
		options.Limit,
		options.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list student class histories: %w", err)
	}
	defer rows.Close()

	histories := make([]domain.StudentClassHistory, 0)

	for rows.Next() {
		history, err := scanStudentClassHistory(rows)
		if err != nil {
			return nil, fmt.Errorf("scan student class history: %w", err)
		}

		histories = append(histories, history)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate student class histories: %w", err)
	}

	return histories, nil
}

func (r *StudentClassHistoryRepository) GetCurrentByStudentAndAcademicYear(
	ctx context.Context,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	dateValue time.Time,
) (domain.StudentClassHistory, error) {
	const query = `
		SELECT
			id,
			student_id,
			academic_year_id,
			class_id,
			lower(validity),
			upper(validity),
			created_at,
			updated_at
		FROM student_class_histories
		WHERE student_id = $1
			AND academic_year_id = $2
			AND validity @> $3::date
		ORDER BY lower(validity) DESC, id DESC
		LIMIT 1
	`

	return r.getOne(
		ctx,
		query,
		studentID,
		academicYearID,
		dateValue,
	)
}

func (r *StudentClassHistoryRepository) getOne(
	ctx context.Context,
	query string,
	args ...any,
) (domain.StudentClassHistory, error) {
	var history domain.StudentClassHistory
	var upper sql.NullTime

	err := r.db.QueryRowContext(
		ctx,
		query,
		args...,
	).Scan(
		&history.ID,
		&history.StudentID,
		&history.AcademicYearID,
		&history.ClassID,
		&history.Period.From,
		&upper,
		&history.CreatedAt,
		&history.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.StudentClassHistory{}, repository.ErrNotFound
		}

		return domain.StudentClassHistory{}, fmt.Errorf(
			"get student class history: %w",
			err,
		)
	}

	if upper.Valid {
		history.Period.To = &upper.Time
	}

	return history, nil
}

func scanStudentClassHistory(rows *sql.Rows) (domain.StudentClassHistory, error) {
	var history domain.StudentClassHistory
	var upper sql.NullTime

	err := rows.Scan(
		&history.ID,
		&history.StudentID,
		&history.AcademicYearID,
		&history.ClassID,
		&history.Period.From,
		&upper,
		&history.CreatedAt,
		&history.UpdatedAt,
	)
	if err != nil {
		return domain.StudentClassHistory{}, err
	}

	if upper.Valid {
		history.Period.To = &upper.Time
	}

	return history, nil
}

func translateStudentClassHistoryError(operation string, err error) error {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "23P01":
			return fmt.Errorf("%s: %w", operation, repository.ErrConflict)
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}

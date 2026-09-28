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

var _ repository.AcademicYearRepository = (*AcademicYearRepository)(nil)

type AcademicYearRepository struct {
	db database.DBTX
}

func NewAcademicYearRepository(db database.DBTX) *AcademicYearRepository {
	return &AcademicYearRepository{
		db: db,
	}
}

func (r *AcademicYearRepository) Create(
	ctx context.Context,
	academicYear domain.AcademicYear,
) error {
	const query = `
		INSERT INTO academic_years (
			id,
			name,
			start_date,
			end_date,
			status,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7
		)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		academicYear.ID,
		academicYear.Name,
		academicYear.StartDate,
		academicYear.EndDate,
		academicYear.Status,
		academicYear.CreatedAt,
		academicYear.UpdatedAt,
	)
	if err != nil {
		return translateError("create academic year", err)
	}

	return nil
}

func (r *AcademicYearRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.AcademicYear, error) {
	const query = `
		SELECT
			id,
			name,
			start_date,
			end_date,
			status,
			created_at,
			updated_at
		FROM academic_years
		WHERE id = $1
	`

	return r.getOne(ctx, query, id, "get academic year by id")
}

func (r *AcademicYearRepository) List(
	ctx context.Context,
	options repository.ListOptions,
) ([]domain.AcademicYear, error) {
	const query = `
		SELECT
			id,
			name,
			start_date,
			end_date,
			status,
			created_at,
			updated_at
		FROM academic_years
		ORDER BY start_date DESC, id DESC
		LIMIT $1
		OFFSET $2
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		options.Limit,
		options.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list academic years: %w", err)
	}
	defer rows.Close()

	academicYears := make([]domain.AcademicYear, 0)

	for rows.Next() {
		var academicYear domain.AcademicYear

		if err := rows.Scan(
			&academicYear.ID,
			&academicYear.Name,
			&academicYear.StartDate,
			&academicYear.EndDate,
			&academicYear.Status,
			&academicYear.CreatedAt,
			&academicYear.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan academic year: %w", err)
		}

		academicYears = append(academicYears, academicYear)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate academic years: %w", err)
	}

	return academicYears, nil
}

func (r *AcademicYearRepository) Update(
	ctx context.Context,
	academicYear domain.AcademicYear,
) error {
	const query = `
		UPDATE academic_years
		SET
			name = $1,
			start_date = $2,
			end_date = $3,
			status = $4,
			updated_at = $5
		WHERE id = $6
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		academicYear.Name,
		academicYear.StartDate,
		academicYear.EndDate,
		academicYear.Status,
		academicYear.UpdatedAt,
		academicYear.ID,
	)
	if err != nil {
		return translateError("update academic year", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"get updated academic year rows affected: %w",
			err,
		)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r *AcademicYearRepository) GetOpen(
	ctx context.Context,
) (domain.AcademicYear, error) {
	const query = `
		SELECT
			id,
			name,
			start_date,
			end_date,
			status,
			created_at,
			updated_at
		FROM academic_years
		WHERE status = 'OPEN'
	`

	return r.getOne(ctx, query, nil, "get open academic year")
}

func (r *AcademicYearRepository) getOne(
	ctx context.Context,
	query string,
	arg any,
	operation string,
) (domain.AcademicYear, error) {
	var academicYear domain.AcademicYear

	var err error
	if arg == nil {
		err = r.db.QueryRowContext(ctx, query).Scan(
			&academicYear.ID,
			&academicYear.Name,
			&academicYear.StartDate,
			&academicYear.EndDate,
			&academicYear.Status,
			&academicYear.CreatedAt,
			&academicYear.UpdatedAt,
		)
	} else {
		err = r.db.QueryRowContext(ctx, query, arg).Scan(
			&academicYear.ID,
			&academicYear.Name,
			&academicYear.StartDate,
			&academicYear.EndDate,
			&academicYear.Status,
			&academicYear.CreatedAt,
			&academicYear.UpdatedAt,
		)
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AcademicYear{}, repository.ErrNotFound
		}

		return domain.AcademicYear{}, fmt.Errorf(
			"%s: %w",
			operation,
			err,
		)
	}

	return academicYear, nil
}

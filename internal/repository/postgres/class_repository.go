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

var _ repository.ClassRepository = (*ClassRepository)(nil)

type ClassRepository struct {
	db database.DBTX
}

func NewClassRepository(db database.DBTX) *ClassRepository {
	return &ClassRepository{
		db: db,
	}
}

func (r *ClassRepository) Create(
	ctx context.Context,
	class domain.Class,
) error {
	const query = `
		INSERT INTO classes (
			id,
			name,
			level,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5
		)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		class.ID,
		class.Name,
		class.Level,
		class.CreatedAt,
		class.UpdatedAt,
	)
	if err != nil {
		return translateError("create class", err)
	}

	return nil
}

func (r *ClassRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.Class, error) {
	const query = `
		SELECT
			id,
			name,
			level,
			created_at,
			updated_at
		FROM classes
		WHERE id = $1
	`

	var class domain.Class

	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
		&class.ID,
		&class.Name,
		&class.Level,
		&class.CreatedAt,
		&class.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Class{}, repository.ErrNotFound
		}

		return domain.Class{}, fmt.Errorf(
			"get class by id: %w",
			err,
		)
	}

	return class, nil
}

func (r *ClassRepository) List(
	ctx context.Context,
	options repository.ListOptions,
) ([]domain.Class, error) {
	const query = `
		SELECT
			id,
			name,
			level,
			created_at,
			updated_at
		FROM classes
		ORDER BY level ASC, name ASC, id ASC
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
		return nil, fmt.Errorf("list classes: %w", err)
	}
	defer rows.Close()

	classes := make([]domain.Class, 0)

	for rows.Next() {
		var class domain.Class

		if err := rows.Scan(
			&class.ID,
			&class.Name,
			&class.Level,
			&class.CreatedAt,
			&class.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan class: %w", err)
		}

		classes = append(classes, class)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate classes: %w", err)
	}

	return classes, nil
}

func (r *ClassRepository) ListByTeacherAndAcademicYear(
	ctx context.Context,
	userID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.Class, error) {
	const query = `
		SELECT
			c.id,
			c.name,
			c.level,
			c.created_at,
			c.updated_at
		FROM classes c
		INNER JOIN teacher_class_assignments tca
			ON tca.class_id = c.id
			AND tca.academic_year_id = $2
		WHERE tca.user_id = $1
		ORDER BY c.level ASC, c.name ASC, c.id ASC
		LIMIT $3
		OFFSET $4
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		userID,
		academicYearID,
		options.Limit,
		options.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list classes by teacher and academic year: %w", err)
	}
	defer rows.Close()

	classes := make([]domain.Class, 0)

	for rows.Next() {
		var class domain.Class

		if err := rows.Scan(
			&class.ID,
			&class.Name,
			&class.Level,
			&class.CreatedAt,
			&class.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan class by teacher and academic year: %w", err)
		}

		classes = append(classes, class)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate classes by teacher and academic year: %w",
			err,
		)
	}

	return classes, nil
}

func (r *ClassRepository) Update(
	ctx context.Context,
	class domain.Class,
) error {
	const query = `
		UPDATE classes
		SET
			name = $1,
			level = $2,
			updated_at = $3
		WHERE id = $4
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		class.Name,
		class.Level,
		class.UpdatedAt,
		class.ID,
	)
	if err != nil {
		return translateError("update class", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"get updated class rows affected: %w",
			err,
		)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r *ClassRepository) ExistsByNameAndLevel(
	ctx context.Context,
	name string,
	level int,
) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM classes
			WHERE name = $1
			  AND level = $2
		)
	`

	var exists bool

	if err := r.db.QueryRowContext(
		ctx,
		query,
		name,
		level,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf(
			"check class existence: %w",
			err,
		)
	}

	return exists, nil
}

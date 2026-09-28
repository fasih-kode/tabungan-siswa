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
		ORDER BY name ASC, id ASC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
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

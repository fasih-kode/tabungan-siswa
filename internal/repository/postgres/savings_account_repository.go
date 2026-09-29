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
	"github.com/jackc/pgx/v5/pgconn"
)

var _ repository.SavingsAccountRepository = (*SavingsAccountRepository)(nil)

type SavingsAccountRepository struct {
	db database.DBTX
}

func NewSavingsAccountRepository(db database.DBTX) *SavingsAccountRepository {
	return &SavingsAccountRepository{
		db: db,
	}
}

func (r *SavingsAccountRepository) Create(
	ctx context.Context,
	account domain.SavingsAccount,
) error {
	const query = `
		INSERT INTO savings_accounts (
			id,
			student_id,
			academic_year_id,
			status,
			settled_at,
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
		account.ID,
		account.StudentID,
		account.AcademicYearID,
		account.Status,
		account.SettledAt,
		account.CreatedAt,
		account.UpdatedAt,
	)
	if err != nil {
		return translateSavingsAccountError(
			"create savings account",
			err,
		)
	}

	return nil
}

func (r *SavingsAccountRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.SavingsAccount, error) {
	const query = `
		SELECT
			id,
			student_id,
			academic_year_id,
			status,
			settled_at,
			created_at,
			updated_at
		FROM savings_accounts
		WHERE id = $1
	`

	return r.getOne(ctx, query, id)
}

func (r *SavingsAccountRepository) GetByStudentAndAcademicYear(
	ctx context.Context,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
) (domain.SavingsAccount, error) {
	const query = `
		SELECT
			id,
			student_id,
			academic_year_id,
			status,
			settled_at,
			created_at,
			updated_at
		FROM savings_accounts
		WHERE student_id = $1
			AND academic_year_id = $2
	`

	return r.getOne(
		ctx,
		query,
		studentID,
		academicYearID,
	)
}

func (r *SavingsAccountRepository) GetByIDForUpdate(
	ctx context.Context,
	id uuid.UUID,
) (domain.SavingsAccount, error) {
	const query = `
		SELECT
			id,
			student_id,
			academic_year_id,
			status,
			settled_at,
			created_at,
			updated_at
		FROM savings_accounts
		WHERE id = $1
		FOR UPDATE
	`

	return r.getOne(ctx, query, id)
}

func (r *SavingsAccountRepository) Update(
	ctx context.Context,
	account domain.SavingsAccount,
) error {
	const query = `
		UPDATE savings_accounts
		SET
			student_id = $2,
			academic_year_id = $3,
			status = $4,
			settled_at = $5,
			updated_at = $6
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		account.ID,
		account.StudentID,
		account.AcademicYearID,
		account.Status,
		account.SettledAt,
		account.UpdatedAt,
	)
	if err != nil {
		return translateSavingsAccountError(
			"update savings account",
			err,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"get updated savings account rows affected: %w",
			err,
		)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r *SavingsAccountRepository) getOne(
	ctx context.Context,
	query string,
	args ...any,
) (domain.SavingsAccount, error) {
	var account domain.SavingsAccount

	err := r.db.QueryRowContext(
		ctx,
		query,
		args...,
	).Scan(
		&account.ID,
		&account.StudentID,
		&account.AcademicYearID,
		&account.Status,
		&account.SettledAt,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.SavingsAccount{}, repository.ErrNotFound
		}

		return domain.SavingsAccount{}, fmt.Errorf(
			"get savings account: %w",
			err,
		)
	}

	return account, nil
}

func translateSavingsAccountError(
	operation string,
	err error,
) error {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "23P01":
			return fmt.Errorf(
				"%s: %w",
				operation,
				repository.ErrConflict,
			)
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}

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

var _ repository.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	db database.DBTX
}

func NewUserRepository(db database.DBTX) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) Create(
	ctx context.Context,
	user domain.User,
) error {
	const query = `
		INSERT INTO users (
			id,
			username,
			password_hash,
			role,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6
		)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		user.ID,
		user.Username,
		user.PasswordHash,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return translateError("create user", err)
	}

	return nil
}

func (r *UserRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.User, error) {
	const query = `
		SELECT
			id,
			username,
			password_hash,
			role,
			created_at,
			updated_at
		FROM users
		WHERE id = $1
	`

	return r.getOne(ctx, query, id, "get user by id")
}

func (r *UserRepository) GetByUsername(
	ctx context.Context,
	username string,
) (domain.User, error) {
	const query = `
		SELECT
			id,
			username,
			password_hash,
			role,
			created_at,
			updated_at
		FROM users
		WHERE username = $1
	`

	return r.getOne(ctx, query, username, "get user by username")
}

func (r *UserRepository) Update(
	ctx context.Context,
	user domain.User,
) error {
	const query = `
		UPDATE users
		SET
			username = $1,
			password_hash = $2,
			role = $3,
			updated_at = $4
		WHERE id = $5
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		user.Username,
		user.PasswordHash,
		user.Role,
		user.UpdatedAt,
		user.ID,
	)
	if err != nil {
		return translateError("update user", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get updated user rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r *UserRepository) ExistsByUsername(
	ctx context.Context,
	username string,
) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM users
			WHERE username = $1
		)
	`

	var exists bool

	if err := r.db.QueryRowContext(
		ctx,
		query,
		username,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf(
			"check user username existence: %w",
			err,
		)
	}

	return exists, nil
}

func (r *UserRepository) getOne(
	ctx context.Context,
	query string,
	arg any,
	operation string,
) (domain.User, error) {
	var user domain.User

	err := r.db.QueryRowContext(
		ctx,
		query,
		arg,
	).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, repository.ErrNotFound
		}

		return domain.User{}, fmt.Errorf("%s: %w", operation, err)
	}

	return user, nil
}

func translateError(operation string, err error) error {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return fmt.Errorf(
				"%s: %w",
				operation,
				repository.ErrConflict,
			)
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}

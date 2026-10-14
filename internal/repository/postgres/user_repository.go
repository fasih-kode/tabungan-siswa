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

func (r *UserRepository) ListByRole(
	ctx context.Context,
	role domain.UserRole,
	options repository.ListOptions,
) ([]domain.User, error) {
	const query = `
		SELECT
			id,
			username,
			role,
			created_at,
			updated_at
		FROM users
		WHERE role = $1
		ORDER BY username ASC, id ASC
		LIMIT $2
		OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, role, options.Limit, options.Offset)
	if err != nil {
		return nil, fmt.Errorf("list users by role: %w", err)
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user by role: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users by role: %w", err)
	}

	return users, nil
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
		if pgErr.Code == "23505" || pgErr.Code == "23P01" {
			return fmt.Errorf("%s: %w", operation, repository.ErrConflict)
		}
		if pgErr.Code == "23514" {
			switch pgErr.ConstraintName {
			case "academic_years_close_requires_settled_accounts":
				return fmt.Errorf("%s: %w", operation, domain.ErrInvalidTransition)
			case "academic_years_history_bounds_check":
				return fmt.Errorf("%s: %w", operation, domain.ErrInvalidDateRange)
			}
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}

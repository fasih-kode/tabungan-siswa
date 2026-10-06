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

var _ repository.SessionRepository = (*SessionRepository)(nil)

type SessionRepository struct {
	db database.DBTX
}

func NewSessionRepository(db database.DBTX) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(
	ctx context.Context,
	session domain.Session,
) error {
	const query = `
		INSERT INTO sessions (
			id,
			user_id,
			token_hash,
			expires_at,
			revoked_at,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		session.ID,
		session.UserID,
		session.TokenHash,
		session.ExpiresAt,
		session.RevokedAt,
		session.CreatedAt,
	)
	if err != nil {
		return translateSessionError("create session", err)
	}

	return nil
}

func (r *SessionRepository) GetByTokenHash(
	ctx context.Context,
	tokenHash string,
) (domain.Session, error) {
	const query = `
		SELECT
			id,
			user_id,
			token_hash,
			expires_at,
			revoked_at,
			created_at
		FROM sessions
		WHERE token_hash = $1
	`

	var session domain.Session

	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.ExpiresAt,
		&session.RevokedAt,
		&session.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Session{}, repository.ErrNotFound
		}

		return domain.Session{}, fmt.Errorf(
			"get session by token hash: %w",
			err,
		)
	}

	return session, nil
}

func (r *SessionRepository) Revoke(
	ctx context.Context,
	sessionID uuid.UUID,
	revokedAt time.Time,
) error {
	const query = `
		UPDATE sessions
		SET revoked_at = $2
		WHERE id = $1
			AND revoked_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, sessionID, revokedAt)
	if err != nil {
		return translateSessionError("revoke session", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get revoked session rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func translateSessionError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%s: %w", operation, repository.ErrConflict)
	}

	return fmt.Errorf("%s: %w", operation, err)
}

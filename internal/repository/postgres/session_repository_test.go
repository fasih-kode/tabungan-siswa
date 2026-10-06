package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/google/uuid"
)

func TestSessionRepository_CreateAndGetByTokenHash(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSessionRepository(db)
	ctx := context.Background()

	user := newTestUser(t)
	userRepo := postgres.NewUserRepository(db)
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Create user error = %v", err)
	}
	t.Cleanup(func() {
		deleteTestUser(t, db, user.ID)
	})

	session := newTestSession(t, user.ID)
	if err := repo.Create(ctx, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(deleteTestSession(t, db, session.ID))

	got, err := repo.GetByTokenHash(ctx, session.TokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash() error = %v", err)
	}

	assertSessionEqual(t, got, session)
}

func TestSessionRepository_GetByTokenHashNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSessionRepository(db)

	_, err := repo.GetByTokenHash(
		context.Background(),
		"missing-session-token-hash-"+uuid.NewString(),
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("GetByTokenHash() error = %v, want ErrNotFound", err)
	}
}

func TestSessionRepository_CreateDuplicateTokenHash(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSessionRepository(db)
	ctx := context.Background()

	user := newTestUser(t)
	userRepo := postgres.NewUserRepository(db)
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Create user error = %v", err)
	}
	t.Cleanup(func() {
		deleteTestUser(t, db, user.ID)
	})

	first := newTestSession(t, user.ID)
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	t.Cleanup(deleteTestSession(t, db, first.ID))

	second := newTestSession(t, user.ID)
	second.TokenHash = first.TokenHash

	err := repo.Create(ctx, second)
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("second Create() error = %v, want ErrConflict", err)
	}
}

func TestSessionRepository_Revoke(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSessionRepository(db)
	ctx := context.Background()

	user := newTestUser(t)
	userRepo := postgres.NewUserRepository(db)
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Create user error = %v", err)
	}
	t.Cleanup(func() {
		deleteTestUser(t, db, user.ID)
	})

	session := newTestSession(t, user.ID)
	if err := repo.Create(ctx, session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(deleteTestSession(t, db, session.ID))

	revokedAt := time.Now()
	if err := repo.Revoke(ctx, session.ID, revokedAt); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	got, err := repo.GetByTokenHash(ctx, session.TokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash() after Revoke() error = %v", err)
	}

	if got.RevokedAt == nil {
		t.Fatal("RevokedAt = nil, want non-nil")
	}
	if !got.RevokedAt.Truncate(time.Microsecond).Equal(
		revokedAt.Truncate(time.Microsecond),
	) {
		t.Fatalf("RevokedAt = %v, want %v", got.RevokedAt, revokedAt)
	}
}

func TestSessionRepository_RevokeNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSessionRepository(db)

	err := repo.Revoke(
		context.Background(),
		uuid.New(),
		time.Now(),
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("Revoke() error = %v, want ErrNotFound", err)
	}
}

func TestSessionRepository_CreateRejectsUnknownUser(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewSessionRepository(db)

	session := newTestSession(t, uuid.New())
	err := repo.Create(context.Background(), session)
	if err == nil {
		t.Fatal("Create() error = nil, want foreign key error")
	}
}

func newTestSession(t *testing.T, userID uuid.UUID) domain.Session {
	t.Helper()

	session, err := domain.NewSession(
		userID,
		"token-hash-"+uuid.NewString(),
		time.Now().Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	return session
}

func assertSessionEqual(
	t *testing.T,
	got domain.Session,
	want domain.Session,
) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("ID = %v, want %v", got.ID, want.ID)
	}
	if got.UserID != want.UserID {
		t.Errorf("UserID = %v, want %v", got.UserID, want.UserID)
	}
	if got.TokenHash != want.TokenHash {
		t.Errorf("TokenHash = %q, want %q", got.TokenHash, want.TokenHash)
	}
	if !got.ExpiresAt.Truncate(time.Microsecond).Equal(
		want.ExpiresAt.Truncate(time.Microsecond),
	) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, want.ExpiresAt)
	}
	if got.RevokedAt != nil {
		t.Errorf("RevokedAt = %v, want nil", got.RevokedAt)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		if !got.CreatedAt.Truncate(time.Microsecond).Equal(
			want.CreatedAt.Truncate(time.Microsecond),
		) {
			t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, want.CreatedAt)
		}
	}
}

func deleteTestSession(t *testing.T, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, id uuid.UUID) func() {
	t.Helper()
	return func() {
		if _, err := db.ExecContext(context.Background(), "DELETE FROM sessions WHERE id = $1", id); err != nil {
			t.Errorf("delete test session: %v", err)
		}
	}
}

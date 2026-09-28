package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/google/uuid"
)

func TestUserRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewUserRepository(db)
	ctx := context.Background()

	user := newTestUser(t)

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, db, user.ID)
	})

	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertUserEqual(t, got, user)
}

func TestUserRepository_GetByIDNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewUserRepository(db)

	_, err := repo.GetByID(
		context.Background(),
		uuid.New(),
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"GetByID() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestUserRepository_GetByUsername(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewUserRepository(db)
	ctx := context.Background()

	user := newTestUser(t)

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, db, user.ID)
	})

	got, err := repo.GetByUsername(ctx, user.Username)
	if err != nil {
		t.Fatalf("GetByUsername() error = %v", err)
	}

	assertUserEqual(t, got, user)
}

func TestUserRepository_GetByUsernameNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewUserRepository(db)

	_, err := repo.GetByUsername(
		context.Background(),
		"username-does-not-exist-"+uuid.NewString(),
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"GetByUsername() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestUserRepository_ExistsByUsername(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewUserRepository(db)
	ctx := context.Background()

	user := newTestUser(t)

	exists, err := repo.ExistsByUsername(ctx, user.Username)
	if err != nil {
		t.Fatalf("ExistsByUsername() error = %v", err)
	}

	if exists {
		t.Fatal("ExistsByUsername() = true before user exists")
	}

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, db, user.ID)
	})

	exists, err = repo.ExistsByUsername(ctx, user.Username)
	if err != nil {
		t.Fatalf("ExistsByUsername() error = %v", err)
	}

	if !exists {
		t.Fatal("ExistsByUsername() = false after user exists")
	}
}

func TestUserRepository_Update(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewUserRepository(db)
	ctx := context.Background()

	user := newTestUser(t)

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, db, user.ID)
	})

	updated := user
	updated.Username = "updated-" + uuid.NewString()
	updated.PasswordHash = "updated-password-hash"
	updated.Role = domain.RoleSiswa
	updated.UpdatedAt = time.Now()

	if err := repo.Update(ctx, updated); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID() after Update() error = %v", err)
	}

	assertUserEqual(t, got, updated)
}

func TestUserRepository_UpdateNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewUserRepository(db)

	user := newTestUser(t)

	err := repo.Update(
		context.Background(),
		user,
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"Update() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestUserRepository_CreateDuplicateUsername(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewUserRepository(db)
	ctx := context.Background()

	first := newTestUser(t)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, db, first.ID)
	})

	second := newTestUser(t)
	second.Username = first.Username

	err := repo.Create(ctx, second)
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf(
			"second Create() error = %v, want ErrConflict",
			err,
		)
	}
}

func assertUserEqual(
	t *testing.T,
	got domain.User,
	want domain.User,
) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("ID = %v, want %v", got.ID, want.ID)
	}

	if got.Username != want.Username {
		t.Errorf(
			"Username = %q, want %q",
			got.Username,
			want.Username,
		)
	}

	if got.PasswordHash != want.PasswordHash {
		t.Errorf(
			"PasswordHash = %q, want %q",
			got.PasswordHash,
			want.PasswordHash,
		)
	}

	if got.Role != want.Role {
		t.Errorf(
			"Role = %q, want %q",
			got.Role,
			want.Role,
		)
	}

	if !got.CreatedAt.Equal(want.CreatedAt) {
		if !got.CreatedAt.Truncate(time.Microsecond).Equal(
			want.CreatedAt.Truncate(time.Microsecond),
		) {
			t.Errorf(
				"CreatedAt = %v, want %v",
				got.CreatedAt,
				want.CreatedAt,
			)
		}
	}

	if !got.UpdatedAt.Equal(want.UpdatedAt) {
		if !got.UpdatedAt.Truncate(time.Microsecond).Equal(
			want.UpdatedAt.Truncate(time.Microsecond),
		) {
			t.Errorf(
				"UpdatedAt = %v, want %v",
				got.UpdatedAt,
				want.UpdatedAt,
			)
		}
	}
}

func newTestUser(t *testing.T) domain.User {
	t.Helper()

	user, err := domain.NewUser(
		"user-"+uuid.NewString(),
		"test-password-hash",
		domain.RoleAdmin,
	)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}

	return user
}

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL is not set")
	}

	db, err := database.Open(dsn)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}

	if err := database.Ping(context.Background(), db); err != nil {
		_ = db.Close()
		t.Fatalf("database.Ping() error = %v", err)
	}

	t.Cleanup(func() {
		if err := database.Close(db); err != nil {
			t.Errorf("database.Close() error = %v", err)
		}
	})

	return db
}

func deleteTestUser(
	t *testing.T,
	db *sql.DB,
	id uuid.UUID,
) {
	t.Helper()

	_, err := db.ExecContext(
		context.Background(),
		"DELETE FROM users WHERE id = $1",
		id,
	)
	if err != nil {
		t.Errorf("delete test user: %v", err)
	}
}

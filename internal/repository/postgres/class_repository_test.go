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

func TestClassRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassRepository(db)
	ctx := context.Background()

	class := newTestClass(t, "7A", 7)

	if err := repo.Create(ctx, class); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestClass(t, db, class.ID)
	})

	got, err := repo.GetByID(ctx, class.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	assertClassEqual(t, got, class)
}

func TestClassRepository_GetByIDNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassRepository(db)

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

func TestClassRepository_List(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassRepository(db)
	ctx := context.Background()

	cleanupClasses(t, db)

	first := newTestClass(t, "B", 7)
	second := newTestClass(t, "A", 7)
	third := newTestClass(t, "A", 8)

	for _, class := range []domain.Class{
		first,
		second,
		third,
	} {
		if err := repo.Create(ctx, class); err != nil {
			t.Fatalf("Create(%s) error = %v", class.Name, err)
		}
	}

	t.Cleanup(func() {
		deleteTestClass(t, db, first.ID)
		deleteTestClass(t, db, second.ID)
		deleteTestClass(t, db, third.ID)
	})

	got, err := repo.List(ctx, repository.ListOptions{
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("List() length = %d, want 3", len(got))
	}

	assertClassEqual(t, got[0], second)
	assertClassEqual(t, got[1], first)
	assertClassEqual(t, got[2], third)
}

func TestClassRepository_ListPagination(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassRepository(db)
	ctx := context.Background()

	cleanupClasses(t, db)

	first := newTestClass(t, "A", 7)
	second := newTestClass(t, "B", 7)
	third := newTestClass(t, "C", 7)

	for _, class := range []domain.Class{
		first,
		second,
		third,
	} {
		if err := repo.Create(ctx, class); err != nil {
			t.Fatalf("Create(%s) error = %v", class.Name, err)
		}
	}

	t.Cleanup(func() {
		deleteTestClass(t, db, first.ID)
		deleteTestClass(t, db, second.ID)
		deleteTestClass(t, db, third.ID)
	})

	got, err := repo.List(ctx, repository.ListOptions{
		Limit:  1,
		Offset: 1,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("List() length = %d, want 1", len(got))
	}

	assertClassEqual(t, got[0], second)
}

func TestClassRepository_Update(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassRepository(db)
	ctx := context.Background()

	class := newTestClass(t, "7A", 7)

	if err := repo.Create(ctx, class); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestClass(t, db, class.ID)
	})

	updated := class
	updated.Name = "7B"
	updated.Level = 8
	updated.UpdatedAt = time.Now()

	if err := repo.Update(ctx, updated); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repo.GetByID(ctx, class.ID)
	if err != nil {
		t.Fatalf("GetByID() after Update() error = %v", err)
	}

	assertClassEqual(t, got, updated)
}

func TestClassRepository_UpdateNotFound(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassRepository(db)

	class := newTestClass(t, "7A", 7)

	err := repo.Update(
		context.Background(),
		class,
	)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"Update() error = %v, want ErrNotFound",
			err,
		)
	}
}

func TestClassRepository_UpdateDuplicateNameAndLevel(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassRepository(db)
	ctx := context.Background()

	cleanupClasses(t, db)

	first := newTestClass(t, "7A", 7)
	second := newTestClass(t, "7B", 7)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("second Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestClass(t, db, first.ID)
		deleteTestClass(t, db, second.ID)
	})

	second.Name = first.Name
	second.Level = first.Level
	second.UpdatedAt = time.Now()

	err := repo.Update(ctx, second)
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf(
			"Update() error = %v, want ErrConflict",
			err,
		)
	}
}

func TestClassRepository_CreateDuplicateNameAndLevel(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassRepository(db)
	ctx := context.Background()

	cleanupClasses(t, db)

	first := newTestClass(t, "7A", 7)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestClass(t, db, first.ID)
	})

	second := newTestClass(t, "7A", 7)

	err := repo.Create(ctx, second)
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf(
			"second Create() error = %v, want ErrConflict",
			err,
		)
	}
}

func TestClassRepository_ExistsByNameAndLevelTrue(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassRepository(db)
	ctx := context.Background()

	class := newTestClass(t, "7A", 7)

	if err := repo.Create(ctx, class); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestClass(t, db, class.ID)
	})

	exists, err := repo.ExistsByNameAndLevel(ctx, "7A", 7)
	if err != nil {
		t.Fatalf("ExistsByNameAndLevel() error = %v", err)
	}

	if !exists {
		t.Fatal("ExistsByNameAndLevel() = false, want true")
	}
}

func TestClassRepository_ExistsByNameAndLevelFalse(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassRepository(db)

	cleanupClasses(t, db)

	exists, err := repo.ExistsByNameAndLevel(
		context.Background(),
		"7A",
		7,
	)
	if err != nil {
		t.Fatalf("ExistsByNameAndLevel() error = %v", err)
	}

	if exists {
		t.Fatal("ExistsByNameAndLevel() = true, want false")
	}
}

func TestClassRepository_ExistsByNameAndLevelDifferentLevel(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassRepository(db)
	ctx := context.Background()

	class := newTestClass(t, "7A", 7)

	if err := repo.Create(ctx, class); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Cleanup(func() {
		deleteTestClass(t, db, class.ID)
	})

	exists, err := repo.ExistsByNameAndLevel(ctx, "7A", 8)
	if err != nil {
		t.Fatalf("ExistsByNameAndLevel() error = %v", err)
	}

	if exists {
		t.Fatal("ExistsByNameAndLevel() = true, want false for different level")
	}
}

func TestClassRepository_ExistsByNameAndLevelDatabaseError(t *testing.T) {
	db := openTestDatabase(t)
	repo := postgres.NewClassRepository(db)

	if err := db.Close(); err != nil {
		t.Fatalf("db.Close() error = %v", err)
	}

	exists, err := repo.ExistsByNameAndLevel(
		context.Background(),
		"7A",
		7,
	)
	if err == nil {
		t.Fatal("ExistsByNameAndLevel() error = nil, want error")
	}

	if exists {
		t.Fatal("ExistsByNameAndLevel() = true, want false")
	}
}

func assertClassEqual(
	t *testing.T,
	got domain.Class,
	want domain.Class,
) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("ID = %v, want %v", got.ID, want.ID)
	}

	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}

	if got.Level != want.Level {
		t.Errorf("Level = %d, want %d", got.Level, want.Level)
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

func newTestClass(
	t *testing.T,
	name string,
	level int,
) domain.Class {
	t.Helper()

	class, err := domain.NewClass(name, level)
	if err != nil {
		t.Fatalf("NewClass() error = %v", err)
	}

	return class
}

func deleteTestClass(
	t *testing.T,
	db *sql.DB,
	id uuid.UUID,
) {
	t.Helper()

	_, err := db.ExecContext(
		context.Background(),
		"DELETE FROM classes WHERE id = $1",
		id,
	)
	if err != nil {
		t.Errorf("delete test class: %v", err)
	}
}

func cleanupClasses(
	t *testing.T,
	db *sql.DB,
) {
	t.Helper()

	_, err := db.ExecContext(
		context.Background(),
		"DELETE FROM classes",
	)
	if err != nil {
		t.Fatalf("cleanup classes: %v", err)
	}
}

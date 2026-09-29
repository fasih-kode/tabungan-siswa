package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
)

func TestPostgresUnitOfWork_BeginProvidesAllRepositories(t *testing.T) {
	db := openTestDatabase(t)
	ctx := context.Background()

	manager := postgres.NewUnitOfWorkManager(db)

	uow, err := manager.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	if err := uow.Rollback(); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	repos := uow.Repositories()

	if repos.Users == nil {
		t.Error("Repositories().Users is nil")
	}
	if repos.AcademicYears == nil {
		t.Error("Repositories().AcademicYears is nil")
	}
	if repos.Classes == nil {
		t.Error("Repositories().Classes is nil")
	}
	if repos.Students == nil {
		t.Error("Repositories().Students is nil")
	}
	if repos.StudentClassHistories == nil {
		t.Error("Repositories().StudentClassHistories is nil")
	}
	if repos.TeacherClassAssignments == nil {
		t.Error("Repositories().TeacherClassAssignments is nil")
	}
	if repos.ClassTransferRequests == nil {
		t.Error("Repositories().ClassTransferRequests is nil")
	}
	if repos.SavingsAccounts == nil {
		t.Error("Repositories().SavingsAccounts is nil")
	}
	if repos.Transactions == nil {
		t.Error("Repositories().Transactions is nil")
	}
	if repos.Settlements == nil {
		t.Error("Repositories().Settlements is nil")
	}
	if repos.AuditLogs == nil {
		t.Error("Repositories().AuditLogs is nil")
	}
}

func TestPostgresUnitOfWork_RollbackRollsBackRepositoryChanges(t *testing.T) {
	db := openTestDatabase(t)
	ctx := context.Background()

	class, err := domain.NewClass(
		fmt.Sprintf("UOW Rollback %s", os.Getenv("USER")),
		1,
	)
	if err != nil {
		t.Fatalf("NewClass() error = %v", err)
	}

	manager := postgres.NewUnitOfWorkManager(db)

	uow, err := manager.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	repos := uow.Repositories()

	if err := repos.Classes.Create(ctx, class); err != nil {
		_ = uow.Rollback()
		t.Fatalf("Classes.Create() error = %v", err)
	}

	if err := uow.Rollback(); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	got, err := postgres.NewClassRepository(db).GetByID(ctx, class.ID)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf(
			"GetByID() error = %v, want ErrNotFound; got = %#v",
			err,
			got,
		)
	}
}

func TestPostgresUnitOfWork_CommitPersistsRepositoryChanges(t *testing.T) {
	db := openTestDatabase(t)
	ctx := context.Background()

	class, err := domain.NewClass(
		fmt.Sprintf("UOW Commit %s", os.Getenv("USER")),
		2,
	)
	if err != nil {
		t.Fatalf("NewClass() error = %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.ExecContext(
			ctx,
			"DELETE FROM classes WHERE id = $1",
			class.ID,
		)
	})

	manager := postgres.NewUnitOfWorkManager(db)

	uow, err := manager.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	repos := uow.Repositories()

	if err := repos.Classes.Create(ctx, class); err != nil {
		_ = uow.Rollback()
		t.Fatalf("Classes.Create() error = %v", err)
	}

	if err := uow.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	got, err := postgres.NewClassRepository(db).GetByID(ctx, class.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != class.ID {
		t.Fatalf("GetByID() ID = %v, want %v", got.ID, class.ID)
	}
}

func TestNewUnitOfWorkManager_NilDatabase(t *testing.T) {
	manager := postgres.NewUnitOfWorkManager(nil)

	uow, err := manager.Begin(context.Background())
	if err == nil {
		t.Fatal("Begin() error = nil, want error")
	}

	if uow != nil {
		t.Fatalf("Begin() uow = %#v, want nil", uow)
	}
}

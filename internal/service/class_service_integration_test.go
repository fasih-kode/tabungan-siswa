package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestClassServiceIntegration_CrossServiceCommit(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()

	cleanupServiceScopeFixtures(t, db)

	userRepo := postgres.NewUserRepository(db)
	classRepo := postgres.NewClassRepository(db)
	auditRepo := postgres.NewAuditRepository(db)
	manager := postgres.NewUnitOfWorkManager(db)

	actorUser := newServiceScopeUser(t, domain.RoleAdmin)
	if err := userRepo.Create(ctx, actorUser); err != nil {
		t.Fatalf("create actor: %v", err)
	}

	svc, err := service.NewClassService(service.Dependencies{
		Repositories: repository.RepositorySet{
			Classes:   classRepo,
			AuditLogs: auditRepo,
		},
		UOW: manager,
	})
	if err != nil {
		t.Fatalf("NewClassService() error = %v", err)
	}

	var classID uuid.UUID

	err = service.WithinTransaction(
		ctx,
		manager,
		func(txCtx context.Context, _ repository.RepositorySet) error {
			output, err := svc.Create(txCtx, service.CreateClassInput{
				Actor: service.Actor{
					UserID: actorUser.ID,
					Role:   domain.RoleAdmin,
				},
				Name:  "Final Audit Commit",
				Level: 7,
			})
			if err != nil {
				return err
			}
			classID = output.Class.ID
			return nil
		},
	)
	if err != nil {
		t.Fatalf("WithinTransaction() error = %v", err)
	}

	class, err := classRepo.GetByID(ctx, classID)
	if err != nil {
		t.Fatalf("GetByID() class after commit: %v", err)
	}
	if class.Name != "Final Audit Commit" || class.Level != 7 {
		t.Fatalf("class = %#v, want Final Audit Commit/7", class)
	}

	audits, err := auditRepo.ListByEntity(
		ctx,
		domain.AuditEntityClass,
		classID,
		repository.ListOptions{Limit: 10},
	)
	if err != nil {
		t.Fatalf("ListByEntity() error = %v", err)
	}
	if len(audits) != 1 {
		t.Fatalf("audit count = %d, want 1", len(audits))
	}
	if audits[0].Action != "CREATE" {
		t.Fatalf("audit action = %q, want CREATE", audits[0].Action)
	}
}

func TestClassServiceIntegration_CrossServiceRollbackIsAtomic(t *testing.T) {
	db := openServiceScopeTestDatabase(t)
	ctx := context.Background()

	cleanupServiceScopeFixtures(t, db)

	userRepo := postgres.NewUserRepository(db)
	classRepo := postgres.NewClassRepository(db)
	auditRepo := postgres.NewAuditRepository(db)
	manager := postgres.NewUnitOfWorkManager(db)

	actorUser := newServiceScopeUser(t, domain.RoleAdmin)
	if err := userRepo.Create(ctx, actorUser); err != nil {
		t.Fatalf("create actor: %v", err)
	}

	svc, err := service.NewClassService(service.Dependencies{
		Repositories: repository.RepositorySet{
			Classes:   classRepo,
			AuditLogs: auditRepo,
		},
		UOW: manager,
	})
	if err != nil {
		t.Fatalf("NewClassService() error = %v", err)
	}

	classID := uuid.Nil
	wantErr := errors.New("force class rollback")

	err = service.WithinTransaction(
		ctx,
		manager,
		func(txCtx context.Context, _ repository.RepositorySet) error {
			output, err := svc.Create(txCtx, service.CreateClassInput{
				Actor: service.Actor{
					UserID: actorUser.ID,
					Role:   domain.RoleAdmin,
				},
				Name:  "Final Audit Rollback",
				Level: 7,
			})
			if err != nil {
				return err
			}
			classID = output.Class.ID
			return wantErr
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("WithinTransaction() error = %v, want %v", err, wantErr)
	}

	if _, err := classRepo.GetByID(ctx, classID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("GetByID() class error = %v, want repository.ErrNotFound", err)
	}

	audits, err := auditRepo.ListByEntity(
		ctx,
		domain.AuditEntityClass,
		classID,
		repository.ListOptions{Limit: 10},
	)
	if err != nil {
		t.Fatalf("ListByEntity() error = %v", err)
	}
	if len(audits) != 0 {
		t.Fatalf("audit count = %d, want 0 after rollback", len(audits))
	}
}

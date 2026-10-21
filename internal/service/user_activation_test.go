package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

func TestUserServiceSetActiveRequiresAdminAndProtectsAdminAccounts(t *testing.T) {
	ctx := context.Background()
	adminID, targetID := uuid.New(), uuid.New()
	admin := Actor{UserID: adminID, Role: domain.RoleAdmin}
	userRepo := &userServiceFakeUserRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.User, error) {
			return newTestDomainUser(t, targetID, "target", domain.RoleSiswa), nil
		},
	}
	uow := &userServiceFakeUnitOfWork{repos: repository.RepositorySet{
		Users:    &userServiceFakeActivationRepository{UserRepository: userRepo},
		Sessions: &userServiceFakeSessionRevoker{},
	}}
	svc := newUserServiceForTest(t, repository.RepositorySet{}, &userServiceFakeUnitOfWorkManager{uow: uow})

	tests := []struct {
		name  string
		actor Actor
		id    uuid.UUID
		role  domain.UserRole
		want  error
	}{
		{name: "non admin", actor: Actor{UserID: uuid.New(), Role: domain.RoleWaliKelas}, id: targetID, role: domain.RoleSiswa, want: ErrForbidden},
		{name: "self deactivation", actor: admin, id: adminID, role: domain.RoleSiswa, want: ErrForbidden},
		{name: "admin account", actor: admin, id: targetID, role: domain.RoleAdmin, want: ErrForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userRepo.getByIDFn = func(context.Context, uuid.UUID) (domain.User, error) {
				return newTestDomainUser(t, tc.id, "target", tc.role), nil
			}
			err := svc.SetActive(ctx, SetUserActiveInput{Actor: tc.actor, UserID: tc.id, Active: false})
			if !errors.Is(err, tc.want) {
				t.Fatalf("SetActive() error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestUserServiceSetActiveRollsBackWhenSessionRevocationFails(t *testing.T) {
	target := newTestDomainUser(t, uuid.New(), "target", domain.RoleSiswa)
	revokeErr := errors.New("session store unavailable")
	userRepo := &userServiceFakeUserRepository{getByIDFn: func(context.Context, uuid.UUID) (domain.User, error) { return target, nil }}
	uow := &userServiceFakeUnitOfWork{repos: repository.RepositorySet{
		Users:    &userServiceFakeActivationRepository{UserRepository: userRepo},
		Sessions: &userServiceFakeSessionRevoker{revokeAllFn: func(context.Context, uuid.UUID, time.Time) error { return revokeErr }},
	}}
	svc := newUserServiceForTest(t, repository.RepositorySet{}, &userServiceFakeUnitOfWorkManager{uow: uow})
	err := svc.SetActive(context.Background(), SetUserActiveInput{Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, UserID: target.ID, Active: false})
	if !errors.Is(err, revokeErr) {
		t.Fatalf("SetActive() error = %v, want %v", err, revokeErr)
	}
	if uow.commitCount != 0 || uow.rollbackCount != 1 {
		t.Fatalf("commits=%d rollbacks=%d", uow.commitCount, uow.rollbackCount)
	}
}

func TestUserServicePasswordResetRollsBackWhenSessionRevocationFails(t *testing.T) {
	target := newTestDomainUser(t, uuid.New(), "target", domain.RoleSiswa)
	revokeErr := errors.New("session store unavailable")
	userRepo := &userServiceFakeUserRepository{getByIDFn: func(context.Context, uuid.UUID) (domain.User, error) { return target, nil }}
	uow := &userServiceFakeUnitOfWork{repos: repository.RepositorySet{
		Users:    userRepo,
		Sessions: &userServiceFakeSessionRevoker{revokeAllFn: func(context.Context, uuid.UUID, time.Time) error { return revokeErr }},
	}}
	svc := newUserServiceForTest(t, repository.RepositorySet{}, &userServiceFakeUnitOfWorkManager{uow: uow})
	_, err := svc.Update(context.Background(), UpdateUserInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, UserID: target.ID,
		Username: target.Username, Role: target.Role, PasswordHash: "new-password-hash",
		PasswordChangeRequired: true, RevokeSessions: true,
	})
	if !errors.Is(err, revokeErr) {
		t.Fatalf("Update() error = %v, want %v", err, revokeErr)
	}
	if uow.commitCount != 0 || uow.rollbackCount != 1 {
		t.Fatalf("commits=%d rollbacks=%d", uow.commitCount, uow.rollbackCount)
	}
}

func TestUserServiceUpdateRejectsRoleMutation(t *testing.T) {
	ctx := context.Background()
	admin := Actor{UserID: uuid.New(), Role: domain.RoleAdmin}
	target := newTestDomainUser(t, uuid.New(), "target", domain.RoleSiswa)
	userRepo := &userServiceFakeUserRepository{getByIDFn: func(context.Context, uuid.UUID) (domain.User, error) { return target, nil }}
	uow := &userServiceFakeUnitOfWork{repos: repository.RepositorySet{Users: userRepo}}
	svc := newUserServiceForTest(t, repository.RepositorySet{}, &userServiceFakeUnitOfWorkManager{uow: uow})
	_, err := svc.Update(ctx, UpdateUserInput{Actor: admin, UserID: target.ID, Username: target.Username, Role: domain.RoleAdmin, PasswordHash: target.PasswordHash})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Update() error = %v, want ErrForbidden", err)
	}
}

func TestUserServiceSetActiveDeactivationRevokesSessions(t *testing.T) {
	ctx := context.Background()
	admin := Actor{UserID: uuid.New(), Role: domain.RoleAdmin}
	target := newTestDomainUser(t, uuid.New(), "target", domain.RoleWaliKelas)
	var statusChanged, sessionsRevoked bool
	userRepo := &userServiceFakeUserRepository{getByIDFn: func(context.Context, uuid.UUID) (domain.User, error) { return target, nil }}
	uow := &userServiceFakeUnitOfWork{repos: repository.RepositorySet{
		Users: &userServiceFakeActivationRepository{UserRepository: userRepo, setActiveFn: func(_ context.Context, id uuid.UUID, active bool, _ time.Time) error {
			statusChanged = id == target.ID && !active
			return nil
		}},
		Sessions: &userServiceFakeSessionRevoker{revokeAllFn: func(_ context.Context, id uuid.UUID, _ time.Time) error {
			sessionsRevoked = id == target.ID
			return nil
		}},
	}}
	svc := newUserServiceForTest(t, repository.RepositorySet{}, &userServiceFakeUnitOfWorkManager{uow: uow})
	if err := svc.SetActive(ctx, SetUserActiveInput{Actor: admin, UserID: target.ID, Active: false}); err != nil {
		t.Fatalf("SetActive() error = %v", err)
	}
	if !statusChanged || !sessionsRevoked || uow.commitCount != 1 {
		t.Fatalf("statusChanged=%v sessionsRevoked=%v commits=%d", statusChanged, sessionsRevoked, uow.commitCount)
	}
}

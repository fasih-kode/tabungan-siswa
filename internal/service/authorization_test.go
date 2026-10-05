package service_test

import (
	"errors"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestRequireRoleAcceptsAllowedRole(t *testing.T) {
	actor := service.Actor{
		UserID: uuid.New(),
		Role:   domain.RoleAdmin,
	}

	if err := service.RequireRole(actor, domain.RoleAdmin); err != nil {
		t.Fatalf("RequireRole() error = %v, want nil", err)
	}
}

func TestRequireRoleAcceptsAnyMatchingRole(t *testing.T) {
	actor := service.Actor{
		UserID: uuid.New(),
		Role:   domain.RoleWaliKelas,
	}

	if err := service.RequireRole(
		actor,
		domain.RoleAdmin,
		domain.RoleWaliKelas,
	); err != nil {
		t.Fatalf("RequireRole() error = %v, want nil", err)
	}
}

func TestRequireRoleRejectsForbiddenRole(t *testing.T) {
	actor := service.Actor{
		UserID: uuid.New(),
		Role:   domain.RoleSiswa,
	}

	err := service.RequireRole(actor, domain.RoleAdmin)
	if !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("RequireRole() error = %v, want ErrForbidden", err)
	}
}

func TestRequireRoleRejectsInvalidActor(t *testing.T) {
	actor := service.Actor{
		Role: domain.RoleAdmin,
	}

	err := service.RequireRole(actor, domain.RoleAdmin)
	if !errors.Is(err, service.ErrInvalidActor) {
		t.Fatalf("RequireRole() error = %v, want ErrInvalidActor", err)
	}
}

func TestRequireRoleRejectsNoAllowedRoles(t *testing.T) {
	actor := service.Actor{
		UserID: uuid.New(),
		Role:   domain.RoleAdmin,
	}

	err := service.RequireRole(actor)
	if !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("RequireRole() error = %v, want ErrForbidden", err)
	}
}

func TestRequireSelfAcceptsOwnUserID(t *testing.T) {
	userID := uuid.New()
	actor := service.Actor{
		UserID: userID,
		Role:   domain.RoleSiswa,
	}

	if err := service.RequireSelf(actor, userID); err != nil {
		t.Fatalf("RequireSelf() error = %v, want nil", err)
	}
}

func TestRequireSelfRejectsDifferentUserID(t *testing.T) {
	actor := service.Actor{
		UserID: uuid.New(),
		Role:   domain.RoleSiswa,
	}

	err := service.RequireSelf(actor, uuid.New())
	if !errors.Is(err, service.ErrScopeViolation) {
		t.Fatalf("RequireSelf() error = %v, want ErrScopeViolation", err)
	}
}

func TestRequireSelfRejectsNilTargetUserID(t *testing.T) {
	actor := service.Actor{
		UserID: uuid.New(),
		Role:   domain.RoleSiswa,
	}

	err := service.RequireSelf(actor, uuid.Nil)
	if !errors.Is(err, service.ErrScopeViolation) {
		t.Fatalf("RequireSelf() error = %v, want ErrScopeViolation", err)
	}
}

func TestRequireSelfRejectsInvalidActor(t *testing.T) {
	actor := service.Actor{
		UserID: uuid.Nil,
		Role:   domain.RoleSiswa,
	}

	err := service.RequireSelf(actor, uuid.New())
	if !errors.Is(err, service.ErrInvalidActor) {
		t.Fatalf("RequireSelf() error = %v, want ErrInvalidActor", err)
	}
}

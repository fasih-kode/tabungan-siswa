package service_test

import (
	"errors"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestActorValidateAcceptsValidRoles(t *testing.T) {
	roles := []domain.UserRole{
		domain.RoleAdmin,
		domain.RoleWaliKelas,
		domain.RoleSiswa,
	}

	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			actor := service.Actor{
				UserID: uuid.New(),
				Role:   role,
			}

			if err := actor.Validate(); err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
		})
	}
}

func TestActorValidateRejectsNilUserID(t *testing.T) {
	actor := service.Actor{
		UserID: uuid.Nil,
		Role:   domain.RoleAdmin,
	}

	err := actor.Validate()
	if !errors.Is(err, service.ErrInvalidActor) {
		t.Fatalf("Validate() error = %v, want ErrInvalidActor", err)
	}
}

func TestActorValidateRejectsInvalidRole(t *testing.T) {
	actor := service.Actor{
		UserID: uuid.New(),
		Role:   domain.UserRole("INVALID"),
	}

	err := actor.Validate()
	if !errors.Is(err, service.ErrInvalidActor) {
		t.Fatalf("Validate() error = %v, want ErrInvalidActor", err)
	}
}

func TestActorValidateRejectsZeroValueActor(t *testing.T) {
	var actor service.Actor

	err := actor.Validate()
	if !errors.Is(err, service.ErrInvalidActor) {
		t.Fatalf("Validate() error = %v, want ErrInvalidActor", err)
	}
}

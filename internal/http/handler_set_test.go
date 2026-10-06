package http

import (
	"context"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type handlerSetAuthenticationServiceFake struct{}

func (handlerSetAuthenticationServiceFake) Authenticate(
	context.Context,
	service.AuthenticateInput,
) (service.AuthenticateOutput, error) {
	return service.AuthenticateOutput{
		Actor: service.Actor{
			UserID: uuid.New(),
			Role:   domain.RoleAdmin,
		},
	}, nil
}

func (handlerSetAuthenticationServiceFake) Logout(
	context.Context,
	service.LogoutInput,
) error {
	return nil
}

type handlerSetSessionRepositoryFake struct{}

func (handlerSetSessionRepositoryFake) Create(
	context.Context,
	domain.Session,
) error {
	return nil
}

func (handlerSetSessionRepositoryFake) GetByTokenHash(
	context.Context,
	string,
) (domain.Session, error) {
	return domain.Session{}, repository.ErrNotFound
}

func (handlerSetSessionRepositoryFake) Revoke(
	context.Context,
	uuid.UUID,
	time.Time,
) error {
	return nil
}

func TestNewHandlerSet(t *testing.T) {
	expiration, err := security.NewSessionExpirationPolicy(time.Hour)
	if err != nil {
		t.Fatalf("NewSessionExpirationPolicy() error = %v", err)
	}

	set, err := NewHandlerSet(
		service.ServiceSet{
			Authentication: handlerSetAuthenticationServiceFake{},
		},
		repository.RepositorySet{
			Sessions: handlerSetSessionRepositoryFake{},
		},
		security.SecuritySet{
			SessionCookie:     security.NewDefaultSessionCookie(),
			SessionExpiration: expiration,
		},
	)
	if err != nil {
		t.Fatalf("NewHandlerSet() error = %v", err)
	}

	if set.Authentication == nil {
		t.Fatal("HandlerSet.Authentication is nil")
	}
}

func TestNewHandlerSetRejectsMissingAuthenticationService(t *testing.T) {
	expiration, err := security.NewSessionExpirationPolicy(time.Hour)
	if err != nil {
		t.Fatalf("NewSessionExpirationPolicy() error = %v", err)
	}

	_, err = NewHandlerSet(
		service.ServiceSet{},
		repository.RepositorySet{
			Sessions: handlerSetSessionRepositoryFake{},
		},
		security.SecuritySet{
			SessionCookie:     security.NewDefaultSessionCookie(),
			SessionExpiration: expiration,
		},
	)
	if err != service.ErrInvalidDependency {
		t.Fatalf(
			"NewHandlerSet() error = %v, want %v",
			err,
			service.ErrInvalidDependency,
		)
	}
}

func TestNewHandlerSetRejectsMissingSessionRepository(t *testing.T) {
	expiration, err := security.NewSessionExpirationPolicy(time.Hour)
	if err != nil {
		t.Fatalf("NewSessionExpirationPolicy() error = %v", err)
	}

	_, err = NewHandlerSet(
		service.ServiceSet{
			Authentication: handlerSetAuthenticationServiceFake{},
		},
		repository.RepositorySet{},
		security.SecuritySet{
			SessionCookie:     security.NewDefaultSessionCookie(),
			SessionExpiration: expiration,
		},
	)
	if err != service.ErrInvalidDependency {
		t.Fatalf(
			"NewHandlerSet() error = %v, want %v",
			err,
			service.ErrInvalidDependency,
		)
	}
}

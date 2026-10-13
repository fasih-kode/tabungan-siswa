package http

import (
	"context"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
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

func (handlerSetAuthenticationServiceFake) CreateSession(
	context.Context,
	service.CreateSessionInput,
) error {
	return nil
}

func (handlerSetAuthenticationServiceFake) Logout(
	context.Context,
	service.LogoutInput,
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
			AcademicYear:   &academicYearServiceFake{},
		},
		security.SecuritySet{
			SessionCookie:     security.NewDefaultSessionCookie(),
			SessionExpiration: expiration,
		},
		newTestTemplate(t),
	)
	if err != nil {
		t.Fatalf("NewHandlerSet() error = %v", err)
	}

	if set.Authentication == nil {
		t.Fatal("HandlerSet.Authentication is nil")
	}

	if set.LoginPage == nil {
		t.Fatal("HandlerSet.LoginPage is nil")
	}
	if set.Dashboard == nil {
		t.Fatal("HandlerSet.Dashboard is nil")
	}
	if set.AcademicYear == nil {
		t.Fatal("HandlerSet.AcademicYear is nil")
	}
}

func TestNewHandlerSetRejectsMissingAuthenticationService(t *testing.T) {
	expiration, err := security.NewSessionExpirationPolicy(time.Hour)
	if err != nil {
		t.Fatalf("NewSessionExpirationPolicy() error = %v", err)
	}

	_, err = NewHandlerSet(
		service.ServiceSet{},
		security.SecuritySet{
			SessionCookie:     security.NewDefaultSessionCookie(),
			SessionExpiration: expiration,
		},
		newTestTemplate(t),
	)
	if err != service.ErrInvalidDependency {
		t.Fatalf(
			"NewHandlerSet() error = %v, want %v",
			err,
			service.ErrInvalidDependency,
		)
	}
}

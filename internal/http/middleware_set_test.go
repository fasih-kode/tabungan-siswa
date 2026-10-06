package http

import (
	"testing"

	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/fasih/tabungan-siswa/internal/security"
)

func TestNewMiddlewareSet(t *testing.T) {
	repositories := postgres.NewRepositorySet(nil)
	securitySet := security.SecuritySet{
		SessionCookie: security.NewDefaultSessionCookie(),
		CSRFCookie:    security.NewDefaultCSRFCookie(),
	}

	middlewares, err := NewMiddlewareSet(repositories, securitySet)
	if err != nil {
		t.Fatalf("NewMiddlewareSet() error = %v", err)
	}

	if middlewares.Authentication == nil {
		t.Fatal("Authentication middleware = nil, want non-nil")
	}

	if middlewares.CSRF == nil {
		t.Fatal("CSRF middleware = nil, want non-nil")
	}
}

func TestNewMiddlewareSetRejectsMissingSessionRepository(t *testing.T) {
	repositories := postgres.NewRepositorySet(nil)
	repositories.Sessions = nil

	securitySet := security.SecuritySet{
		SessionCookie: security.NewDefaultSessionCookie(),
		CSRFCookie:    security.NewDefaultCSRFCookie(),
	}

	if _, err := NewMiddlewareSet(repositories, securitySet); err == nil {
		t.Fatal("NewMiddlewareSet() error = nil, want error")
	}
}

func TestNewMiddlewareSetRejectsMissingUserRepository(t *testing.T) {
	repositories := postgres.NewRepositorySet(nil)
	repositories.Users = nil

	securitySet := security.SecuritySet{
		SessionCookie: security.NewDefaultSessionCookie(),
		CSRFCookie:    security.NewDefaultCSRFCookie(),
	}

	if _, err := NewMiddlewareSet(repositories, securitySet); err == nil {
		t.Fatal("NewMiddlewareSet() error = nil, want error")
	}
}

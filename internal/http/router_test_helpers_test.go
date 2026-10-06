package http

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/google/uuid"
)

type routerSessionRepositoryFake struct{}

func (f *routerSessionRepositoryFake) Create(context.Context, domain.Session) error {
	return nil
}

func (f *routerSessionRepositoryFake) GetByTokenHash(context.Context, string) (domain.Session, error) {
	return domain.Session{}, repository.ErrNotFound
}

func (f *routerSessionRepositoryFake) Revoke(context.Context, uuid.UUID, time.Time) error {
	return nil
}

type routerUserRepositoryFake struct{}

func (f *routerUserRepositoryFake) Create(context.Context, domain.User) error { return nil }
func (f *routerUserRepositoryFake) GetByID(context.Context, uuid.UUID) (domain.User, error) {
	return domain.User{}, repository.ErrNotFound
}
func (f *routerUserRepositoryFake) GetByUsername(context.Context, string) (domain.User, error) {
	return domain.User{}, repository.ErrNotFound
}
func (f *routerUserRepositoryFake) Update(context.Context, domain.User) error { return nil }
func (f *routerUserRepositoryFake) ExistsByUsername(context.Context, string) (bool, error) {
	return false, nil
}

func newTestSessionCookie(t *testing.T) security.SessionCookie {
	t.Helper()
	return security.NewDefaultSessionCookie()
}

func newTestCSRFCookie(t *testing.T) security.CSRFCookie {
	t.Helper()
	return security.NewDefaultCSRFCookie()
}

var _ http.Handler = http.HandlerFunc(nil)

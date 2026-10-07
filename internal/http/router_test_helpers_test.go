package http

import (
	"context"
	"html/template"
	"net/http"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/google/uuid"
)

type routerSessionRepositoryFake struct {
	session domain.Session
	err     error
}

func (f *routerSessionRepositoryFake) Create(context.Context, domain.Session) error {
	return nil
}

func (f *routerSessionRepositoryFake) GetByTokenHash(context.Context, string) (domain.Session, error) {
	return f.session, f.err
}

func (f *routerSessionRepositoryFake) Revoke(context.Context, uuid.UUID, time.Time) error {
	return nil
}

type routerUserRepositoryFake struct {
	user domain.User
	err  error
}

func (f *routerUserRepositoryFake) Create(context.Context, domain.User) error {
	return nil
}

func (f *routerUserRepositoryFake) GetByID(context.Context, uuid.UUID) (domain.User, error) {
	return f.user, f.err
}

func (f *routerUserRepositoryFake) GetByUsername(context.Context, string) (domain.User, error) {
	return domain.User{}, f.err
}

func (f *routerUserRepositoryFake) Update(context.Context, domain.User) error {
	return nil
}

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

func newTestLoginPageHandler(t *testing.T) *LoginPageHandler {
	t.Helper()

	templates := newTestTemplate(t)

	handler, err := NewLoginPageHandler(templates)
	if err != nil {
		t.Fatalf("NewLoginPageHandler() error = %v", err)
	}

	return handler
}

func newTestTemplate(t *testing.T) *template.Template {
	t.Helper()

	templates, err := template.New("auth").Parse(
		`{{define "auth"}}{{template "auth-content" .}}{{end}}{{define "auth-content"}}<html><body>{{.Title}}</body></html>{{end}}`,
	)
	if err != nil {
		t.Fatalf("test template parse error = %v", err)
	}

	return templates
}

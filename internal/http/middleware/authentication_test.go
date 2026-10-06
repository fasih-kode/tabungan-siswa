package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type authenticationSessionRepositoryFake struct {
	session domain.Session
	err     error
}

func (f *authenticationSessionRepositoryFake) Create(context.Context, domain.Session) error {
	return nil
}

func (f *authenticationSessionRepositoryFake) GetByTokenHash(ctx context.Context, tokenHash string) (domain.Session, error) {
	return f.session, f.err
}

func (f *authenticationSessionRepositoryFake) Revoke(context.Context, uuid.UUID, time.Time) error {
	return nil
}

type authenticationUserRepositoryFake struct {
	user domain.User
	err  error
}

func (f *authenticationUserRepositoryFake) Create(context.Context, domain.User) error {
	return nil
}

func (f *authenticationUserRepositoryFake) GetByID(context.Context, uuid.UUID) (domain.User, error) {
	return f.user, f.err
}

func (f *authenticationUserRepositoryFake) GetByUsername(context.Context, string) (domain.User, error) {
	return domain.User{}, nil
}

func (f *authenticationUserRepositoryFake) Update(context.Context, domain.User) error {
	return nil
}

func (f *authenticationUserRepositoryFake) ExistsByUsername(context.Context, string) (bool, error) {
	return false, nil
}

func newAuthenticationMiddlewareForTest(
	t *testing.T,
	sessionRepo repository.SessionRepository,
	userRepo repository.UserRepository,
) *AuthenticationMiddleware {
	t.Helper()

	middleware, err := NewAuthenticationMiddleware(
		security.NewDefaultSessionCookie(),
		sessionRepo,
		userRepo,
	)
	if err != nil {
		t.Fatalf("NewAuthenticationMiddleware() error = %v", err)
	}

	return middleware
}

func TestNewAuthenticationMiddlewareRejectsMissingDependencies(t *testing.T) {
	userRepo := &authenticationUserRepositoryFake{}
	sessionRepo := &authenticationSessionRepositoryFake{}

	if _, err := NewAuthenticationMiddleware(security.NewDefaultSessionCookie(), nil, userRepo); !errors.Is(err, service.ErrInvalidDependency) {
		t.Fatalf("missing session repository error = %v, want %v", err, service.ErrInvalidDependency)
	}

	if _, err := NewAuthenticationMiddleware(security.NewDefaultSessionCookie(), sessionRepo, nil); !errors.Is(err, service.ErrInvalidDependency) {
		t.Fatalf("missing user repository error = %v, want %v", err, service.ErrInvalidDependency)
	}
}

func TestAuthenticationMiddlewareInjectsActor(t *testing.T) {
	userID := uuid.New()
	user := domain.User{ID: userID, Role: domain.RoleAdmin}
	session := domain.Session{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: security.HashSessionToken("opaque-session-token"),
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}

	middleware := newAuthenticationMiddlewareForTest(
		t,
		&authenticationSessionRepositoryFake{session: session},
		&authenticationUserRepositoryFake{user: user},
	)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := service.ActorFromContext(r.Context())
		if !ok {
			t.Fatal("ActorFromContext() ok = false, want true")
		}
		if actor.UserID != userID || actor.Role != domain.RoleAdmin {
			t.Fatalf("actor = %+v, want user ID %v and role %v", actor, userID, domain.RoleAdmin)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  security.DefaultSessionCookieName,
		Value: "opaque-session-token",
	})
	recorder := httptest.NewRecorder()

	middleware.RequireAuthentication(next).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestAuthenticationMiddlewareRejectsMissingCookie(t *testing.T) {
	middleware := newAuthenticationMiddlewareForTest(
		t,
		&authenticationSessionRepositoryFake{},
		&authenticationUserRepositoryFake{},
	)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler called for missing cookie")
	})

	recorder := httptest.NewRecorder()
	middleware.RequireAuthentication(next).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/", nil),
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticationMiddlewareRejectsUnknownSession(t *testing.T) {
	middleware := newAuthenticationMiddlewareForTest(
		t,
		&authenticationSessionRepositoryFake{err: repository.ErrNotFound},
		&authenticationUserRepositoryFake{},
	)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler called for unknown session")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  security.DefaultSessionCookieName,
		Value: "unknown-token",
	})
	recorder := httptest.NewRecorder()

	middleware.RequireAuthentication(next).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticationMiddlewareRejectsExpiredSession(t *testing.T) {
	session := domain.Session{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: security.HashSessionToken("expired-token"),
		ExpiresAt: time.Now().Add(-time.Minute),
		CreatedAt: time.Now().Add(-time.Hour),
	}

	middleware := newAuthenticationMiddlewareForTest(
		t,
		&authenticationSessionRepositoryFake{session: session},
		&authenticationUserRepositoryFake{},
	)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler called for expired session")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  security.DefaultSessionCookieName,
		Value: "expired-token",
	})
	recorder := httptest.NewRecorder()

	middleware.RequireAuthentication(next).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticationMiddlewareRejectsRevokedSession(t *testing.T) {
	revokedAt := time.Now().Add(-time.Minute)
	session := domain.Session{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: security.HashSessionToken("revoked-token"),
		ExpiresAt: time.Now().Add(time.Hour),
		RevokedAt: &revokedAt,
		CreatedAt: time.Now().Add(-time.Hour),
	}

	middleware := newAuthenticationMiddlewareForTest(
		t,
		&authenticationSessionRepositoryFake{session: session},
		&authenticationUserRepositoryFake{},
	)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler called for revoked session")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  security.DefaultSessionCookieName,
		Value: "revoked-token",
	})
	recorder := httptest.NewRecorder()

	middleware.RequireAuthentication(next).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticationMiddlewareRejectsUnknownUser(t *testing.T) {
	session := domain.Session{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: security.HashSessionToken("unknown-user-token"),
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}

	middleware := newAuthenticationMiddlewareForTest(
		t,
		&authenticationSessionRepositoryFake{session: session},
		&authenticationUserRepositoryFake{err: repository.ErrNotFound},
	)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler called for unknown user")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  security.DefaultSessionCookieName,
		Value: "unknown-user-token",
	})
	recorder := httptest.NewRecorder()

	middleware.RequireAuthentication(next).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticationMiddlewarePropagatesRepositoryFailureAsServerError(t *testing.T) {
	middleware := newAuthenticationMiddlewareForTest(
		t,
		&authenticationSessionRepositoryFake{err: errors.New("database unavailable")},
		&authenticationUserRepositoryFake{},
	)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler called for repository failure")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  security.DefaultSessionCookieName,
		Value: "session-token",
	})
	recorder := httptest.NewRecorder()

	middleware.RequireAuthentication(next).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

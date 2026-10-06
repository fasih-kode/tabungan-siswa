package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type authenticationHandlerServiceFake struct {
	authenticateOutput service.AuthenticateOutput
	authenticateErr    error
	logoutErr          error
	logoutInput        service.LogoutInput
}

func (f *authenticationHandlerServiceFake) Authenticate(
	context.Context,
	service.AuthenticateInput,
) (service.AuthenticateOutput, error) {
	return f.authenticateOutput, f.authenticateErr
}

func (f *authenticationHandlerServiceFake) Logout(
	_ context.Context,
	input service.LogoutInput,
) error {
	f.logoutInput = input
	return f.logoutErr
}

type authenticationHandlerSessionRepositoryFake struct {
	created   domain.Session
	createErr error
}

func (f *authenticationHandlerSessionRepositoryFake) Create(
	_ context.Context,
	session domain.Session,
) error {
	f.created = session
	return f.createErr
}

func (f *authenticationHandlerSessionRepositoryFake) GetByTokenHash(context.Context, string) (domain.Session, error) {
	return domain.Session{}, repository.ErrNotFound
}

func (f *authenticationHandlerSessionRepositoryFake) Revoke(context.Context, uuid.UUID, time.Time) error {
	return nil
}

func newAuthenticationHandlerForTest(
	t *testing.T,
	authentication service.AuthenticationService,
	sessions repository.SessionRepository,
) *AuthenticationHandler {
	t.Helper()

	expiration, err := security.NewSessionExpirationPolicy(time.Hour)
	if err != nil {
		t.Fatalf("NewSessionExpirationPolicy() error = %v", err)
	}

	handler, err := NewAuthenticationHandler(
		authentication,
		sessions,
		expiration,
		security.NewDefaultSessionCookie(),
	)
	if err != nil {
		t.Fatalf("NewAuthenticationHandler() error = %v", err)
	}

	return handler
}

func TestNewAuthenticationHandlerRejectsMissingDependencies(t *testing.T) {
	sessions := &authenticationHandlerSessionRepositoryFake{}
	authentication := &authenticationHandlerServiceFake{}
	expiration, err := security.NewSessionExpirationPolicy(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	cookie := security.NewDefaultSessionCookie()

	if _, err := NewAuthenticationHandler(nil, sessions, expiration, cookie); !errors.Is(err, service.ErrInvalidDependency) {
		t.Fatalf("missing authentication service error = %v", err)
	}

	if _, err := NewAuthenticationHandler(authentication, nil, expiration, cookie); !errors.Is(err, service.ErrInvalidDependency) {
		t.Fatalf("missing session repository error = %v", err)
	}
}

func TestAuthenticationHandlerLoginCreatesSessionAndCookie(t *testing.T) {
	userID := uuid.New()
	authentication := &authenticationHandlerServiceFake{
		authenticateOutput: service.AuthenticateOutput{
			Actor: service.Actor{UserID: userID, Role: domain.RoleAdmin},
		},
	}
	sessions := &authenticationHandlerSessionRepositoryFake{}
	handler := newAuthenticationHandlerForTest(t, authentication, sessions)

	form := url.Values{
		"username": {"admin"},
		"password": {"secret"},
	}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	handler.Login(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if sessions.created.UserID != userID {
		t.Fatalf("session user ID = %v, want %v", sessions.created.UserID, userID)
	}
	if sessions.created.TokenHash == "" {
		t.Fatal("session token hash is empty")
	}
	if sessions.created.ExpiresAt.IsZero() {
		t.Fatal("session expiration is zero")
	}

	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("response cookies = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != security.DefaultSessionCookieName {
		t.Fatalf("cookie name = %q, want %q", cookie.Name, security.DefaultSessionCookieName)
	}
	if cookie.Value == "" {
		t.Fatal("session cookie value is empty")
	}
	if security.HashSessionToken(cookie.Value) != sessions.created.TokenHash {
		t.Fatal("cookie token does not match persisted session hash")
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
		t.Fatalf("unexpected session cookie properties: %+v", cookie)
	}
}

func TestAuthenticationHandlerLoginRejectsInvalidCredentials(t *testing.T) {
	authentication := &authenticationHandlerServiceFake{
		authenticateErr: service.ErrInvalidCredentials,
	}
	sessions := &authenticationHandlerSessionRepositoryFake{}
	handler := newAuthenticationHandlerForTest(t, authentication, sessions)

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("username=admin&password=wrong"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	handler.Login(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if sessions.created.ID != uuid.Nil {
		t.Fatal("session was created for invalid credentials")
	}
}

func TestAuthenticationHandlerLoginRejectsNonPost(t *testing.T) {
	handler := newAuthenticationHandlerForTest(
		t,
		&authenticationHandlerServiceFake{},
		&authenticationHandlerSessionRepositoryFake{},
	)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	recorder := httptest.NewRecorder()

	handler.Login(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	if got := recorder.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf("Allow = %q, want %q", got, http.MethodPost)
	}
}

func TestAuthenticationHandlerLoginPropagatesSessionCreateError(t *testing.T) {
	authentication := &authenticationHandlerServiceFake{
		authenticateOutput: service.AuthenticateOutput{
			Actor: service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		},
	}
	sessions := &authenticationHandlerSessionRepositoryFake{
		createErr: errors.New("database unavailable"),
	}
	handler := newAuthenticationHandlerForTest(t, authentication, sessions)

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("username=admin&password=secret"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	handler.Login(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(recorder.Body.String(), "Internal Server Error") {
		t.Fatalf("body = %q, want generic internal error", recorder.Body.String())
	}
}

func TestAuthenticationHandlerLogoutRevokesAndClearsCookie(t *testing.T) {
	authentication := &authenticationHandlerServiceFake{}
	handler := newAuthenticationHandlerForTest(
		t,
		authentication,
		&authenticationHandlerSessionRepositoryFake{},
	)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  security.DefaultSessionCookieName,
		Value: "opaque-session-token",
	})
	recorder := httptest.NewRecorder()

	handler.Logout(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if authentication.logoutInput.SessionTokenHash != security.HashSessionToken("opaque-session-token") {
		t.Fatalf("logout token hash = %q, want hashed token", authentication.logoutInput.SessionTokenHash)
	}

	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("response cookies = %d, want 1", len(cookies))
	}
	if cookies[0].MaxAge != -1 {
		t.Fatalf("cookie MaxAge = %d, want -1", cookies[0].MaxAge)
	}
}

func TestAuthenticationHandlerLogoutWithoutCookieIsIdempotent(t *testing.T) {
	authentication := &authenticationHandlerServiceFake{}
	handler := newAuthenticationHandlerForTest(
		t,
		authentication,
		&authenticationHandlerSessionRepositoryFake{},
	)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	recorder := httptest.NewRecorder()

	handler.Logout(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if authentication.logoutInput.SessionTokenHash != "" {
		t.Fatalf("logout service was called with hash %q, want empty", authentication.logoutInput.SessionTokenHash)
	}
}

func TestAuthenticationHandlerLogoutPropagatesServiceError(t *testing.T) {
	authentication := &authenticationHandlerServiceFake{
		logoutErr: errors.New("database unavailable"),
	}
	handler := newAuthenticationHandlerForTest(
		t,
		authentication,
		&authenticationHandlerSessionRepositoryFake{},
	)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  security.DefaultSessionCookieName,
		Value: "opaque-session-token",
	})
	recorder := httptest.NewRecorder()

	handler.Logout(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if len(recorder.Result().Cookies()) != 0 {
		t.Fatal("session cookie was cleared after logout failure")
	}
}

func TestAuthenticationHandlerLogoutRejectsNonPost(t *testing.T) {
	handler := newAuthenticationHandlerForTest(
		t,
		&authenticationHandlerServiceFake{},
		&authenticationHandlerSessionRepositoryFake{},
	)

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	recorder := httptest.NewRecorder()

	handler.Logout(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	if got := recorder.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf("Allow = %q, want %q", got, http.MethodPost)
	}
}

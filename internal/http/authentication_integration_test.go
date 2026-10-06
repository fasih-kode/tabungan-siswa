package http

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type integrationAuthService struct {
	userID uuid.UUID
	role   domain.UserRole
}

func (s *integrationAuthService) Authenticate(
	context.Context,
	service.AuthenticateInput,
) (service.AuthenticateOutput, error) {
	return service.AuthenticateOutput{
		Actor: service.Actor{
			UserID: s.userID,
			Role:   s.role,
		},
	}, nil
}

func (s *integrationAuthService) Logout(
	_ context.Context,
	input service.LogoutInput,
) error {
	return nil
}

type integrationSessionRepository struct {
	sessions map[string]domain.Session
}

func newIntegrationSessionRepository() *integrationSessionRepository {
	return &integrationSessionRepository{
		sessions: make(map[string]domain.Session),
	}
}

func (r *integrationSessionRepository) Create(
	_ context.Context,
	session domain.Session,
) error {
	r.sessions[session.TokenHash] = session
	return nil
}

func (r *integrationSessionRepository) GetByTokenHash(
	_ context.Context,
	tokenHash string,
) (domain.Session, error) {
	session, ok := r.sessions[tokenHash]
	if !ok {
		return domain.Session{}, repository.ErrNotFound
	}

	return session, nil
}

func (r *integrationSessionRepository) Revoke(
	_ context.Context,
	sessionID uuid.UUID,
	revokedAt time.Time,
) error {
	for hash, session := range r.sessions {
		if session.ID == sessionID {
			session.RevokedAt = &revokedAt
			r.sessions[hash] = session
			return nil
		}
	}

	return repository.ErrNotFound
}

type integrationUserRepository struct {
	user domain.User
}

func (r *integrationUserRepository) Create(context.Context, domain.User) error {
	return nil
}

func (r *integrationUserRepository) GetByID(
	_ context.Context,
	id uuid.UUID,
) (domain.User, error) {
	if id != r.user.ID {
		return domain.User{}, repository.ErrNotFound
	}

	return r.user, nil
}

func (r *integrationUserRepository) GetByUsername(
	context.Context,
	string,
) (domain.User, error) {
	return r.user, nil
}

func (r *integrationUserRepository) Update(context.Context, domain.User) error {
	return nil
}

func (r *integrationUserRepository) ExistsByUsername(context.Context, string) (bool, error) {
	return true, nil
}

func newIntegrationRouter(t *testing.T) *http.ServeMux {
	t.Helper()

	userID := uuid.New()
	sessions := newIntegrationSessionRepository()
	users := &integrationUserRepository{
		user: domain.User{
			ID:       userID,
			Username: "admin",
			Role:     domain.RoleAdmin,
		},
	}

	authService := &integrationAuthService{
		userID: userID,
		role:   domain.RoleAdmin,
	}

	expiration, err := security.NewSessionExpirationPolicy(time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	sessionCookie := security.NewDefaultSessionCookie()
	authHandler, err := NewAuthenticationHandler(
		authService,
		sessions,
		expiration,
		sessionCookie,
	)
	if err != nil {
		t.Fatal(err)
	}

	authMiddleware, err := middleware.NewAuthenticationMiddleware(
		sessionCookie,
		sessions,
		users,
	)
	if err != nil {
		t.Fatal(err)
	}

	csrfMiddleware := middleware.NewCSRFMiddleware(
		security.NewDefaultCSRFCookie(),
	)

	router, err := NewRouter(
		HandlerSet{
			Authentication: authHandler,
		},
		MiddlewareSet{
			Authentication: authMiddleware,
			CSRF:           csrfMiddleware,
		},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := service.ActorFromContext(r.Context())
			if !ok {
				t.Fatal("protected handler missing actor")
			}
			if actor.UserID != userID {
				t.Fatalf("actor user ID = %v, want %v", actor.UserID, userID)
			}
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	return router
}

func TestAuthenticationIntegrationLoginProtectLogout(t *testing.T) {
	router := newIntegrationRouter(t)
	server := httptest.NewServer(router)
	defer server.Close()

	client := server.Client()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client.Jar = jar

	response, err := client.Get(server.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("GET /login status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}

	var csrfToken string
	for _, cookie := range jar.Cookies(response.Request.URL) {
		if cookie.Name == security.DefaultCSRFCookieName {
			csrfToken = cookie.Value
		}
	}
	if csrfToken == "" {
		t.Fatal("GET /login did not establish CSRF cookie")
	}

	loginRequest, err := http.NewRequest(
		http.MethodPost,
		server.URL+"/login",
		strings.NewReader("username=admin&password=secret"),
	)
	if err != nil {
		t.Fatal(err)
	}
	loginRequest.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)
	loginRequest.Header.Set(security.DefaultCSRFHeaderName, csrfToken)

	response, err = client.Do(loginRequest)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("POST /login status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}

	sessionCookieFound := false
	for _, cookie := range jar.Cookies(response.Request.URL) {
		if cookie.Name == security.DefaultSessionCookieName {
			sessionCookieFound = true
			break
		}
	}
	if !sessionCookieFound {
		t.Fatal("POST /login did not establish session cookie")
	}

	protectedRequest, err := http.NewRequest(
		http.MethodGet,
		server.URL+"/protected",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	response, err = client.Do(protectedRequest)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("GET /protected status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}

	logoutRequest, err := http.NewRequest(
		http.MethodPost,
		server.URL+"/logout",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	logoutRequest.Header.Set(security.DefaultCSRFHeaderName, csrfToken)

	response, err = client.Do(logoutRequest)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("POST /logout status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}

	protectedRequest, err = http.NewRequest(
		http.MethodGet,
		server.URL+"/protected",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	response, err = client.Do(protectedRequest)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /protected after logout status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	}
}

func TestAuthenticationIntegrationRejectsLoginWithoutCSRF(t *testing.T) {
	router := newIntegrationRouter(t)
	server := httptest.NewServer(router)
	defer server.Close()

	request, err := http.NewRequest(http.MethodPost, server.URL+"/login", nil)
	if err != nil {
		t.Fatal(err)
	}

	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("POST /login without CSRF status = %d, want %d", response.StatusCode, http.StatusForbidden)
	}
}

func TestAuthenticationIntegrationRejectsProtectedPostBeforeCSRFWhenUnauthenticated(t *testing.T) {
	router := newIntegrationRouter(t)
	server := httptest.NewServer(router)
	defer server.Close()

	request, err := http.NewRequest(http.MethodPost, server.URL+"/protected", nil)
	if err != nil {
		t.Fatal(err)
	}

	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("POST /protected unauthenticated status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	}
}

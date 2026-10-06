package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type realAuthenticationUserRepository struct {
	user domain.User
}

func (r *realAuthenticationUserRepository) Create(context.Context, domain.User) error {
	return nil
}

func (r *realAuthenticationUserRepository) GetByID(
	_ context.Context,
	id uuid.UUID,
) (domain.User, error) {
	if id != r.user.ID {
		return domain.User{}, repository.ErrNotFound
	}

	return r.user, nil
}

func (r *realAuthenticationUserRepository) GetByUsername(
	_ context.Context,
	username string,
) (domain.User, error) {
	if username != r.user.Username {
		return domain.User{}, repository.ErrNotFound
	}

	return r.user, nil
}

func (r *realAuthenticationUserRepository) Update(context.Context, domain.User) error {
	return nil
}

func (r *realAuthenticationUserRepository) ExistsByUsername(
	_ context.Context,
	username string,
) (bool, error) {
	return username == r.user.Username, nil
}

type realAuthenticationSessionRepository struct {
	mu       sync.Mutex
	sessions map[string]domain.Session
}

func newRealAuthenticationSessionRepository() *realAuthenticationSessionRepository {
	return &realAuthenticationSessionRepository{
		sessions: make(map[string]domain.Session),
	}
}

func (r *realAuthenticationSessionRepository) Create(
	_ context.Context,
	session domain.Session,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessions[session.TokenHash] = session
	return nil
}

func (r *realAuthenticationSessionRepository) GetByTokenHash(
	_ context.Context,
	tokenHash string,
) (domain.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[tokenHash]
	if !ok {
		return domain.Session{}, repository.ErrNotFound
	}

	return session, nil
}

func (r *realAuthenticationSessionRepository) Revoke(
	_ context.Context,
	sessionID uuid.UUID,
	revokedAt time.Time,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for tokenHash, session := range r.sessions {
		if session.ID != sessionID || session.RevokedAt != nil {
			continue
		}

		session.RevokedAt = &revokedAt
		r.sessions[tokenHash] = session
		return nil
	}

	return repository.ErrNotFound
}

type realAuthenticationUOW struct{}

func (realAuthenticationUOW) Begin(context.Context) (repository.UnitOfWork, error) {
	return nil, nil
}

func TestAuthenticationIntegrationUsesRealServiceAndArgon2id(t *testing.T) {
	hasher, err := security.NewDefaultArgon2idHasher()
	if err != nil {
		t.Fatalf("NewDefaultArgon2idHasher() error = %v", err)
	}

	password := "correct horse battery staple"

	passwordHash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	user, err := domain.NewUser(
		"admin",
		passwordHash,
		domain.RoleAdmin,
	)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}

	users := &realAuthenticationUserRepository{user: user}
	sessions := newRealAuthenticationSessionRepository()

	authentication, err := service.NewAuthenticationService(
		service.Dependencies{
			Repositories: repository.RepositorySet{
				Users:    users,
				Sessions: sessions,
			},
			UOW:            realAuthenticationUOW{},
			PasswordHasher: hasher,
		},
	)
	if err != nil {
		t.Fatalf("NewAuthenticationService() error = %v", err)
	}

	expiration, err := security.NewSessionExpirationPolicy(time.Hour)
	if err != nil {
		t.Fatalf("NewSessionExpirationPolicy() error = %v", err)
	}

	sessionCookie := security.NewDefaultSessionCookie()

	authenticationHandler, err := NewAuthenticationHandler(
		authentication,
		sessions,
		expiration,
		sessionCookie,
	)
	if err != nil {
		t.Fatalf("NewAuthenticationHandler() error = %v", err)
	}

	authMiddleware, err := middleware.NewAuthenticationMiddleware(
		sessionCookie,
		sessions,
		users,
	)
	if err != nil {
		t.Fatalf("NewAuthenticationMiddleware() error = %v", err)
	}

	csrfMiddleware := middleware.NewCSRFMiddleware(
		security.NewDefaultCSRFCookie(),
	)

	router, err := NewRouter(
		RouterDependencies{
			AuthenticationHandler: authenticationHandler,
			Authentication:        authMiddleware,
			CSRF:                  csrfMiddleware,
		},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	server := httptest.NewServer(router)
	defer server.Close()

	client, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New() error = %v", err)
	}

	httpClient := &http.Client{Jar: client}

	getLogin, err := httpClient.Get(server.URL + "/login")
	if err != nil {
		t.Fatalf("GET /login error = %v", err)
	}
	getLogin.Body.Close()

	if getLogin.StatusCode != http.StatusNoContent {
		t.Fatalf(
			"GET /login status = %d, want %d",
			getLogin.StatusCode,
			http.StatusNoContent,
		)
	}

	csrfCookie := findCookie(
		client,
		server.URL,
		security.DefaultCSRFCookieName,
	)
	if csrfCookie == nil {
		t.Fatalf(
			"CSRF cookie %q was not stored",
			security.DefaultCSRFCookieName,
		)
	}

	form := url.Values{
		"username": {user.Username},
		"password": {password},
	}

	loginRequest, err := http.NewRequest(
		http.MethodPost,
		server.URL+"/login",
		bytes.NewBufferString(form.Encode()),
	)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	loginRequest.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)
	loginRequest.Header.Set("X-CSRF-Token", csrfCookie.Value)

	loginResponse, err := httpClient.Do(loginRequest)
	if err != nil {
		t.Fatalf("POST /login error = %v", err)
	}
	loginResponse.Body.Close()

	if loginResponse.StatusCode != http.StatusNoContent {
		t.Fatalf(
			"POST /login status = %d, want %d",
			loginResponse.StatusCode,
			http.StatusNoContent,
		)
	}

	storedSessionCookie := findCookie(
		client,
		server.URL,
		security.DefaultSessionCookieName,
	)
	if storedSessionCookie == nil {
		t.Fatalf(
			"session cookie %q was not stored",
			security.DefaultSessionCookieName,
		)
	}

	if len(sessions.sessions) != 1 {
		t.Fatalf(
			"stored sessions = %d, want 1",
			len(sessions.sessions),
		)
	}

	protectedRequest, err := http.NewRequest(
		http.MethodGet,
		server.URL+"/protected",
		nil,
	)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	protectedResponse, err := httpClient.Do(protectedRequest)
	if err != nil {
		t.Fatalf("GET /protected error = %v", err)
	}
	protectedResponse.Body.Close()

	if protectedResponse.StatusCode != http.StatusNoContent {
		t.Fatalf(
			"GET /protected status = %d, want %d",
			protectedResponse.StatusCode,
			http.StatusNoContent,
		)
	}

	logoutRequest, err := http.NewRequest(
		http.MethodPost,
		server.URL+"/logout",
		nil,
	)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	logoutRequest.Header.Set("X-CSRF-Token", csrfCookie.Value)

	logoutResponse, err := httpClient.Do(logoutRequest)
	if err != nil {
		t.Fatalf("POST /logout error = %v", err)
	}
	logoutResponse.Body.Close()

	if logoutResponse.StatusCode != http.StatusNoContent {
		t.Fatalf(
			"POST /logout status = %d, want %d",
			logoutResponse.StatusCode,
			http.StatusNoContent,
		)
	}

	protectedAfterLogout, err := http.NewRequest(
		http.MethodGet,
		server.URL+"/protected",
		nil,
	)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	protectedAfterLogoutResponse, err := httpClient.Do(protectedAfterLogout)
	if err != nil {
		t.Fatalf(
			"GET /protected after logout error = %v",
			err,
		)
	}
	protectedAfterLogoutResponse.Body.Close()

	if protectedAfterLogoutResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf(
			"GET /protected after logout status = %d, want %d",
			protectedAfterLogoutResponse.StatusCode,
			http.StatusUnauthorized,
		)
	}
}

func findCookie(
	jar http.CookieJar,
	rawURL string,
	name string,
) *http.Cookie {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}

	for _, cookie := range jar.Cookies(u) {
		if cookie.Name == name {
			copied := *cookie
			return &copied
		}
	}

	return nil
}

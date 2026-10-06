package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestNewRouterRejectsMissingDependencies(t *testing.T) {
	cookie := newTestSessionCookie(t)
	sessionRepo := &routerSessionRepositoryFake{}
	userRepo := &routerUserRepositoryFake{}

	authMiddleware, err := middleware.NewAuthenticationMiddleware(
		cookie,
		sessionRepo,
		userRepo,
	)
	if err != nil {
		t.Fatal(err)
	}

	csrfMiddleware := middleware.NewCSRFMiddleware(
		newTestCSRFCookie(t),
	)

	protected := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	handlers := HandlerSet{
		Authentication: &AuthenticationHandler{},
	}
	middlewares := MiddlewareSet{
		Authentication: authMiddleware,
		CSRF:           csrfMiddleware,
	}

	if _, err := NewRouter(handlers, middlewares, protected); err != nil {
		t.Fatalf("unexpected error = %v", err)
	}

	handlers.Authentication = nil
	if _, err := NewRouter(handlers, middlewares, protected); err != ErrInvalidRouterDependency {
		t.Fatalf("missing handler error = %v, want %v", err, ErrInvalidRouterDependency)
	}
}

func TestNewRouterPublicLoginRoute(t *testing.T) {
	cookie := newTestSessionCookie(t)
	authMiddleware, err := middleware.NewAuthenticationMiddleware(
		cookie,
		&routerSessionRepositoryFake{},
		&routerUserRepositoryFake{},
	)
	if err != nil {
		t.Fatal(err)
	}

	csrfMiddleware := middleware.NewCSRFMiddleware(
		newTestCSRFCookie(t),
	)

	router, err := NewRouter(
		HandlerSet{
			Authentication: &AuthenticationHandler{},
		},
		MiddlewareSet{
			Authentication: authMiddleware,
			CSRF:           csrfMiddleware,
		},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}

	if len(recorder.Result().Cookies()) != 1 {
		t.Fatalf(
			"cookie count = %d, want %d",
			len(recorder.Result().Cookies()),
			1,
		)
	}
}

func TestNewRouterAuthenticationRoutesRequireCSRF(t *testing.T) {
	cookie := newTestSessionCookie(t)
	authMiddleware, err := middleware.NewAuthenticationMiddleware(
		cookie,
		&routerSessionRepositoryFake{},
		&routerUserRepositoryFake{},
	)
	if err != nil {
		t.Fatal(err)
	}

	router, err := NewRouter(
		HandlerSet{
			Authentication: &AuthenticationHandler{},
		},
		MiddlewareSet{
			Authentication: authMiddleware,
			CSRF: middleware.NewCSRFMiddleware(
				newTestCSRFCookie(t),
			),
		},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
	}{
		{
			name: "login",
			path: "/login",
		},
		{
			name: "logout",
			path: "/logout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusForbidden {
				t.Fatalf(
					"POST %s status = %d, want %d",
					tt.path,
					recorder.Code,
					http.StatusForbidden,
				)
			}
		})
	}
}

func TestNewRouterAuthenticationRoutesRejectUnsupportedMethods(t *testing.T) {
	cookie := newTestSessionCookie(t)
	authMiddleware, err := middleware.NewAuthenticationMiddleware(
		cookie,
		&routerSessionRepositoryFake{},
		&routerUserRepositoryFake{},
	)
	if err != nil {
		t.Fatal(err)
	}

	router, err := NewRouter(
		HandlerSet{
			Authentication: &AuthenticationHandler{},
		},
		MiddlewareSet{
			Authentication: authMiddleware,
			CSRF:           middleware.NewCSRFMiddleware(newTestCSRFCookie(t)),
		},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		path      string
		wantAllow string
	}{
		{
			name:      "login",
			path:      "/login",
			wantAllow: "GET, HEAD, POST",
		},
		{
			name:      "logout",
			path:      "/logout",
			wantAllow: "POST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, tt.path, nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusMethodNotAllowed {
				t.Fatalf(
					"PUT %s status = %d, want %d",
					tt.path,
					recorder.Code,
					http.StatusMethodNotAllowed,
				)
			}

			if got := recorder.Header().Get("Allow"); got != tt.wantAllow {
				t.Fatalf(
					"PUT %s Allow = %q, want %q",
					tt.path,
					got,
					tt.wantAllow,
				)
			}
		})
	}
}

func TestNewRouterLogoutRouteUsesPostMethod(t *testing.T) {
	cookie := newTestSessionCookie(t)
	authMiddleware, err := middleware.NewAuthenticationMiddleware(
		cookie,
		&routerSessionRepositoryFake{},
		&routerUserRepositoryFake{},
	)
	if err != nil {
		t.Fatal(err)
	}

	router, err := NewRouter(
		HandlerSet{
			Authentication: &AuthenticationHandler{},
		},
		MiddlewareSet{
			Authentication: authMiddleware,
			CSRF: middleware.NewCSRFMiddleware(
				newTestCSRFCookie(t),
			),
		},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"GET /logout status = %d, want %d",
			recorder.Code,
			http.StatusMethodNotAllowed,
		)
	}

	if got := recorder.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf("Allow = %q, want %q", got, http.MethodPost)
	}
}

func TestNewRouterProtectedRouteRequiresAuthentication(t *testing.T) {
	cookie := newTestSessionCookie(t)
	authMiddleware, err := middleware.NewAuthenticationMiddleware(
		cookie,
		&routerSessionRepositoryFake{},
		&routerUserRepositoryFake{},
	)
	if err != nil {
		t.Fatal(err)
	}

	router, err := NewRouter(
		HandlerSet{
			Authentication: &AuthenticationHandler{},
		},
		MiddlewareSet{
			Authentication: authMiddleware,
			CSRF: middleware.NewCSRFMiddleware(
				newTestCSRFCookie(t),
			),
		},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestNewRouterProtectedRouteAllowsAuthenticatedRequest(t *testing.T) {
	userID := uuid.New()
	sessionToken := "protected-route-session-token"

	session := domain.Session{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: security.HashSessionToken(sessionToken),
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}

	user := domain.User{
		ID:   userID,
		Role: domain.RoleAdmin,
	}

	authMiddleware, err := middleware.NewAuthenticationMiddleware(
		security.NewDefaultSessionCookie(),
		&routerSessionRepositoryFake{session: session},
		&routerUserRepositoryFake{user: user},
	)
	if err != nil {
		t.Fatal(err)
	}

	protectedCalled := false
	protected := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		protectedCalled = true

		actor, ok := service.ActorFromContext(r.Context())
		if !ok {
			t.Fatal("ActorFromContext() ok = false, want true")
		}

		if actor.UserID != userID || actor.Role != domain.RoleAdmin {
			t.Fatalf(
				"actor = %+v, want user ID %v and role %v",
				actor,
				userID,
				domain.RoleAdmin,
			)
		}

		w.WriteHeader(http.StatusNoContent)
	})

	router, err := NewRouter(
		HandlerSet{
			Authentication: &AuthenticationHandler{},
		},
		MiddlewareSet{
			Authentication: authMiddleware,
			CSRF: middleware.NewCSRFMiddleware(
				newTestCSRFCookie(t),
			),
		},
		protected,
	)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  security.DefaultSessionCookieName,
		Value: sessionToken,
	})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}

	if !protectedCalled {
		t.Fatal("protected handler was not called")
	}
}

func TestNewRouterProtectedPostRequiresAuthenticationBeforeCSRF(t *testing.T) {
	cookie := newTestSessionCookie(t)
	authMiddleware, err := middleware.NewAuthenticationMiddleware(
		cookie,
		&routerSessionRepositoryFake{},
		&routerUserRepositoryFake{},
	)
	if err != nil {
		t.Fatal(err)
	}

	router, err := NewRouter(
		HandlerSet{
			Authentication: &AuthenticationHandler{},
		},
		MiddlewareSet{
			Authentication: authMiddleware,
			CSRF: middleware.NewCSRFMiddleware(
				newTestCSRFCookie(t),
			),
		},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/protected", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

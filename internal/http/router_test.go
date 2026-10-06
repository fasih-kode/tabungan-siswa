package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/http/middleware"
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

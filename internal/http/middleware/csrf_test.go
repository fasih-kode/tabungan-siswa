package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/security"
)

func TestCSRFMiddlewareSetsTokenOnSafeRequest(t *testing.T) {
	middleware := NewCSRFMiddleware(security.NewDefaultCSRFCookie())

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	middleware.Protect(next).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/", nil),
	)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if recorder.Header().Get("Set-Cookie") == "" {
		t.Fatal("safe request did not establish CSRF cookie")
	}
}

func TestCSRFMiddlewareDoesNotReplaceExistingToken(t *testing.T) {
	middleware := NewCSRFMiddleware(security.NewDefaultCSRFCookie())

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  security.DefaultCSRFCookieName,
		Value: "existing-token",
	})
	recorder := httptest.NewRecorder()

	middleware.Protect(next).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if recorder.Header().Get("Set-Cookie") != "" {
		t.Fatal("existing CSRF token was replaced")
	}
}

func TestCSRFMiddlewareAllowsValidMutation(t *testing.T) {
	middleware := NewCSRFMiddleware(security.NewDefaultCSRFCookie())

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  security.DefaultCSRFCookieName,
		Value: "valid-token",
	})
	req.Header.Set(security.DefaultCSRFHeaderName, "valid-token")
	recorder := httptest.NewRecorder()

	middleware.Protect(next).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestCSRFMiddlewareRejectsMissingCookie(t *testing.T) {
	middleware := NewCSRFMiddleware(security.NewDefaultCSRFCookie())

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler called without CSRF cookie")
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(security.DefaultCSRFHeaderName, "token")
	recorder := httptest.NewRecorder()

	middleware.Protect(next).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddlewareRejectsMissingHeader(t *testing.T) {
	middleware := NewCSRFMiddleware(security.NewDefaultCSRFCookie())

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler called without CSRF header")
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  security.DefaultCSRFCookieName,
		Value: "valid-token",
	})
	recorder := httptest.NewRecorder()

	middleware.Protect(next).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddlewareRejectsMismatchedToken(t *testing.T) {
	middleware := NewCSRFMiddleware(security.NewDefaultCSRFCookie())

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler called with mismatched CSRF token")
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  security.DefaultCSRFCookieName,
		Value: "cookie-token",
	})
	req.Header.Set(security.DefaultCSRFHeaderName, "different-token")
	recorder := httptest.NewRecorder()

	middleware.Protect(next).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddlewareAllowsSafeMethodsWithoutHeader(t *testing.T) {
	middleware := NewCSRFMiddleware(security.NewDefaultCSRFCookie())

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			recorder := httptest.NewRecorder()

			middleware.Protect(next).ServeHTTP(recorder, req)

			if recorder.Code != http.StatusNoContent {
				t.Fatalf("%s status = %d, want %d", method, recorder.Code, http.StatusNoContent)
			}
		})
	}
}

func TestCSRFMiddlewareRejectsOtherMutationMethodsWithoutToken(t *testing.T) {
	middleware := NewCSRFMiddleware(security.NewDefaultCSRFCookie())

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler called without CSRF token")
	})

	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			recorder := httptest.NewRecorder()

			middleware.Protect(next).ServeHTTP(recorder, req)

			if recorder.Code != http.StatusForbidden {
				t.Fatalf("%s status = %d, want %d", method, recorder.Code, http.StatusForbidden)
			}
		})
	}
}

package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"

	"github.com/fasih/tabungan-siswa/internal/security"
)

type csrfTokenContextKey struct{}

type CSRFMiddleware struct {
	cookie security.CSRFCookie
}

func NewCSRFMiddleware(cookie security.CSRFCookie) *CSRFMiddleware {
	return &CSRFMiddleware{
		cookie: cookie,
	}
}

func (m *CSRFMiddleware) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSafeCSRFMethod(r.Method) {
			token, ok := m.ensureToken(w, r)
			if !ok {
				return
			}

			ctx := context.WithValue(r.Context(), csrfTokenContextKey{}, token)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		cookieToken, err := m.cookie.Read(r)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		requestToken := r.Header.Get(security.DefaultCSRFHeaderName)
		if requestToken == "" {
			requestToken = r.FormValue("csrf_token")
		}

		if requestToken == "" ||
			subtle.ConstantTimeCompare([]byte(cookieToken), []byte(requestToken)) != 1 {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *CSRFMiddleware) ensureToken(w http.ResponseWriter, r *http.Request) (string, bool) {
	if token, err := m.cookie.Read(r); err == nil && token != "" {
		return token, true
	}

	token, err := security.GenerateCSRFToken()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return "", false
	}

	if err := m.cookie.Set(w, token); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return "", false
	}

	return token, true
}

func CSRFToken(r *http.Request) (string, bool) {
	token, ok := r.Context().Value(csrfTokenContextKey{}).(string)
	return token, ok
}

func isSafeCSRFMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

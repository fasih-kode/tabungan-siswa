package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/fasih/tabungan-siswa/internal/security"
)

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
			if !m.ensureToken(w, r) {
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		cookieToken, err := m.cookie.Read(r)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		headerToken := r.Header.Get(security.DefaultCSRFHeaderName)
		if headerToken == "" ||
			subtle.ConstantTimeCompare([]byte(cookieToken), []byte(headerToken)) != 1 {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *CSRFMiddleware) ensureToken(w http.ResponseWriter, r *http.Request) bool {
	if token, err := m.cookie.Read(r); err == nil && token != "" {
		return true
	}

	token, err := security.GenerateCSRFToken()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return false
	}

	if err := m.cookie.Set(w, token); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return false
	}

	return true
}

func isSafeCSRFMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

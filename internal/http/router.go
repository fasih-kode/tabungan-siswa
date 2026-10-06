package http

import (
	"net/http"

	"github.com/fasih/tabungan-siswa/internal/http/middleware"
)

type RouterDependencies struct {
	AuthenticationHandler *AuthenticationHandler
	Authentication        *middleware.AuthenticationMiddleware
	CSRF                  *middleware.CSRFMiddleware
}

func NewRouter(deps RouterDependencies, protected http.Handler) (*http.ServeMux, error) {
	if deps.AuthenticationHandler == nil ||
		deps.Authentication == nil ||
		deps.CSRF == nil ||
		protected == nil {
		return nil, ErrInvalidRouterDependency
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		deps.CSRF.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})).ServeHTTP(w, r)
	})

	mux.HandleFunc("POST /login", deps.CSRF.Protect(http.HandlerFunc(
		deps.AuthenticationHandler.Login,
	)).ServeHTTP)

	mux.HandleFunc("POST /logout", deps.CSRF.Protect(http.HandlerFunc(
		deps.AuthenticationHandler.Logout,
	)).ServeHTTP)

	protectedHandler := deps.Authentication.RequireAuthentication(
		deps.CSRF.Protect(protected),
	)
	mux.Handle("/protected", protectedHandler)

	return mux, nil
}

package http

import (
	"net/http"

	"github.com/fasih/tabungan-siswa/internal/http/middleware"
)

func NewRouter(
	handlers HandlerSet,
	middlewares MiddlewareSet,
	protected http.Handler,
) (*http.ServeMux, error) {
	if handlers.Authentication == nil ||
		middlewares.Authentication == nil ||
		middlewares.CSRF == nil ||
		protected == nil {
		return nil, ErrInvalidRouterDependency
	}

	mux := http.NewServeMux()

	registerRoutes(
		mux,
		handlers,
		middlewares,
		protected,
	)

	return mux, nil
}

func registerRoutes(
	mux *http.ServeMux,
	handlers HandlerSet,
	middlewares MiddlewareSet,
	protected http.Handler,
) {
	registerPublicRoutes(mux, middlewares.CSRF)
	registerAuthenticationRoutes(mux, handlers.Authentication, middlewares.CSRF)
	registerProtectedRoutes(
		mux,
		middlewares.Authentication,
		middlewares.CSRF,
		protected,
	)
}

func registerProtectedRoutes(
	mux *http.ServeMux,
	authentication *middleware.AuthenticationMiddleware,
	csrf *middleware.CSRFMiddleware,
	protected http.Handler,
) {
	protectedHandler := authentication.RequireAuthentication(
		csrf.Protect(protected),
	)
	mux.Handle("/protected", protectedHandler)
}

func registerAuthenticationRoutes(
	mux *http.ServeMux,
	handler *AuthenticationHandler,
	csrf *middleware.CSRFMiddleware,
) {
	mux.HandleFunc("POST /login", csrf.Protect(http.HandlerFunc(
		handler.Login,
	)).ServeHTTP)

	mux.HandleFunc("POST /logout", csrf.Protect(http.HandlerFunc(
		handler.Logout,
	)).ServeHTTP)
}

func registerPublicRoutes(
	mux *http.ServeMux,
	csrf *middleware.CSRFMiddleware,
) {
	mux.HandleFunc("GET /login", csrf.Protect(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	)).ServeHTTP)
}

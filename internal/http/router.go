package http

import "net/http"

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

	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		middlewares.CSRF.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})).ServeHTTP(w, r)
	})

	mux.HandleFunc("POST /login", middlewares.CSRF.Protect(http.HandlerFunc(
		handlers.Authentication.Login,
	)).ServeHTTP)

	mux.HandleFunc("POST /logout", middlewares.CSRF.Protect(http.HandlerFunc(
		handlers.Authentication.Logout,
	)).ServeHTTP)

	protectedHandler := middlewares.Authentication.RequireAuthentication(
		middlewares.CSRF.Protect(protected),
	)
	mux.Handle("/protected", protectedHandler)

	return mux, nil
}

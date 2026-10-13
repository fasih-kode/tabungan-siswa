package http

import (
	"net/http"

	"github.com/fasih/tabungan-siswa/internal/http/middleware"
)

func NewRouter(
	handlers HandlerSet,
	middlewares MiddlewareSet,
	staticAssets http.Handler,
) (*http.ServeMux, error) {
	if handlers.Authentication == nil ||
		handlers.LoginPage == nil ||
		handlers.Dashboard == nil ||
		handlers.AcademicYear == nil ||
		middlewares.Authentication == nil ||
		middlewares.CSRF == nil ||
		staticAssets == nil {
		return nil, ErrInvalidRouterDependency
	}

	mux := http.NewServeMux()

	registerRoutes(
		mux,
		handlers,
		middlewares,
		staticAssets,
	)

	return mux, nil
}

func registerRoutes(
	mux *http.ServeMux,
	handlers HandlerSet,
	middlewares MiddlewareSet,
	staticAssets http.Handler,
) {
	registerStaticRoutes(mux, staticAssets)
	registerPublicRoutes(mux, handlers.LoginPage, middlewares.CSRF)
	registerAuthenticationRoutes(mux, handlers.Authentication, middlewares.CSRF)
	registerAcademicYearRoutes(mux, handlers.AcademicYear, middlewares.Authentication, middlewares.CSRF)
	registerProtectedRoutes(
		mux,
		middlewares.Authentication,
		middlewares.CSRF,
		handlers.Dashboard,
	)
}

func registerStaticRoutes(
	mux *http.ServeMux,
	staticAssets http.Handler,
) {
	mux.Handle(staticRoutePrefix, staticAssets)
}

func registerProtectedRoutes(
	mux *http.ServeMux,
	authentication *middleware.AuthenticationMiddleware,
	csrf *middleware.CSRFMiddleware,
	dashboard *DashboardPageHandler,
) {
	protectedHandler := authentication.RequireAuthentication(
		csrf.Protect(http.HandlerFunc(dashboard.Get)),
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
	loginPage *LoginPageHandler,
	csrf *middleware.CSRFMiddleware,
) {
	mux.HandleFunc("GET /login", csrf.Protect(http.HandlerFunc(
		loginPage.Get,
	)).ServeHTTP)
}

func registerAcademicYearRoutes(
	mux *http.ServeMux,
	handler *AcademicYearPageHandler,
	authentication *middleware.AuthenticationMiddleware,
	csrf *middleware.CSRFMiddleware,
) {
	mux.Handle("GET /academic-years", authentication.RequireAuthentication(csrf.Protect(http.HandlerFunc(handler.List))))
	mux.Handle("POST /academic-years", authentication.RequireAuthentication(limitAcademicYearRequestBody(csrf.Protect(http.HandlerFunc(handler.Create)))))
	mux.Handle("GET /academic-years/{id}", authentication.RequireAuthentication(csrf.Protect(http.HandlerFunc(handler.Detail))))
	mux.Handle("POST /academic-years/{id}/{action}", authentication.RequireAuthentication(limitAcademicYearRequestBody(csrf.Protect(http.HandlerFunc(handler.Transition)))))
}

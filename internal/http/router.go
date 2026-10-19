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
		handlers.Class == nil ||
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
	registerClassRoutes(mux, handlers.Class, middlewares.Authentication, middlewares.CSRF)
	if handlers.Student != nil {
		registerStudentRoutes(mux, handlers.Student, middlewares.Authentication, middlewares.CSRF)
	}
	if handlers.StudentPromotion != nil {
		mux.Handle("GET /student-promotions", middlewares.Authentication.RequireAuthentication(middlewares.CSRF.Protect(http.HandlerFunc(handlers.StudentPromotion.Get))))
		mux.Handle("POST /student-promotions", middlewares.Authentication.RequireAuthentication(limitPromotionRequestBody(middlewares.CSRF.Protect(http.HandlerFunc(handlers.StudentPromotion.Process)))))
	}
	if handlers.ClassTransfer != nil {
		registerClassTransferRoutes(mux, handlers.ClassTransfer, middlewares.Authentication, middlewares.CSRF)
	}
	if handlers.TeacherAssignment != nil {
		registerTeacherAssignmentRoutes(mux, handlers.TeacherAssignment, middlewares.Authentication, middlewares.CSRF)
	}
	if handlers.Users != nil {
		registerUserRoutes(mux, handlers.Users, middlewares.Authentication, middlewares.CSRF)
	}
	registerProtectedRoutes(
		mux,
		middlewares.Authentication,
		middlewares.CSRF,
		handlers.Dashboard,
	)
}

func registerStudentRoutes(mux *http.ServeMux, handler *StudentPageHandler, authentication *middleware.AuthenticationMiddleware, csrf *middleware.CSRFMiddleware) {
	mux.Handle("GET /students/class-options", authentication.RequireAuthentication(http.HandlerFunc(handler.ClassOptions)))
	mux.Handle("GET /students", authentication.RequireAuthentication(csrf.Protect(http.HandlerFunc(handler.List))))
	mux.Handle("POST /students", authentication.RequireAuthentication(limitAcademicYearRequestBody(csrf.Protect(http.HandlerFunc(handler.Create)))))
	mux.Handle("POST /students/import", authentication.RequireAuthentication(limitStudentImportRequestBody(csrf.Protect(http.HandlerFunc(handler.Import)))))
	mux.Handle("GET /students/import/template", authentication.RequireAuthentication(http.HandlerFunc(handler.ImportTemplate)))
	mux.Handle("GET /students/{id}", authentication.RequireAuthentication(csrf.Protect(http.HandlerFunc(handler.Detail))))
	mux.Handle("GET /students/{id}/class-history", authentication.RequireAuthentication(csrf.Protect(http.HandlerFunc(handler.Detail))))
	mux.Handle("GET /students/{id}/edit", authentication.RequireAuthentication(csrf.Protect(http.HandlerFunc(handler.Edit))))
	mux.Handle("POST /students/{id}", authentication.RequireAuthentication(limitAcademicYearRequestBody(csrf.Protect(http.HandlerFunc(handler.Update)))))
	mux.Handle("POST /students/{id}/class-assignments", authentication.RequireAuthentication(limitAcademicYearRequestBody(csrf.Protect(http.HandlerFunc(handler.AssignClass)))))
	mux.Handle("POST /students/{id}/leave", authentication.RequireAuthentication(limitAcademicYearRequestBody(csrf.Protect(http.HandlerFunc(handler.Leave)))))
	mux.Handle("POST /students/{id}/reactivate", authentication.RequireAuthentication(limitAcademicYearRequestBody(csrf.Protect(http.HandlerFunc(handler.Reactivate)))))
}

func registerClassTransferRoutes(
	mux *http.ServeMux,
	handler *ClassTransferPageHandler,
	authentication *middleware.AuthenticationMiddleware,
	csrf *middleware.CSRFMiddleware,
) {
	mux.Handle("GET /students/{student_id}/class-transfer-requests", authentication.RequireAuthentication(csrf.Protect(http.HandlerFunc(handler.ListStudent))))
	mux.Handle("GET /class-transfer-requests/pending", authentication.RequireAuthentication(csrf.Protect(http.HandlerFunc(handler.ListPending))))
	mux.Handle("GET /class-transfer-requests/{id}", authentication.RequireAuthentication(csrf.Protect(http.HandlerFunc(handler.Detail))))
	mux.Handle("POST /class-transfer-requests", authentication.RequireAuthentication(limitAcademicYearRequestBody(csrf.Protect(http.HandlerFunc(handler.Request)))))
	mux.Handle("POST /class-transfer-requests/{id}/approve", authentication.RequireAuthentication(limitAcademicYearRequestBody(csrf.Protect(http.HandlerFunc(handler.Approve)))))
	mux.Handle("POST /class-transfer-requests/{id}/reject", authentication.RequireAuthentication(limitAcademicYearRequestBody(csrf.Protect(http.HandlerFunc(handler.Reject)))))
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

func registerClassRoutes(
	mux *http.ServeMux,
	handler *ClassPageHandler,
	authentication *middleware.AuthenticationMiddleware,
	csrf *middleware.CSRFMiddleware,
) {
	mux.Handle("GET /classes", authentication.RequireAuthentication(csrf.Protect(http.HandlerFunc(handler.List))))
	mux.Handle("POST /classes", authentication.RequireAuthentication(limitAcademicYearRequestBody(csrf.Protect(http.HandlerFunc(handler.Create)))))
	mux.Handle("GET /classes/{id}", authentication.RequireAuthentication(csrf.Protect(http.HandlerFunc(handler.Detail))))
	mux.Handle("POST /classes/{id}", authentication.RequireAuthentication(limitAcademicYearRequestBody(csrf.Protect(http.HandlerFunc(handler.Update)))))
}

func registerTeacherAssignmentRoutes(
	mux *http.ServeMux,
	handler *TeacherAssignmentPageHandler,
	authentication *middleware.AuthenticationMiddleware,
	csrf *middleware.CSRFMiddleware,
) {
	mux.Handle("GET /teacher-assignments", authentication.RequireAuthentication(csrf.Protect(http.HandlerFunc(handler.List))))
	mux.Handle("POST /teacher-assignments", authentication.RequireAuthentication(limitAcademicYearRequestBody(csrf.Protect(http.HandlerFunc(handler.Create)))))
}

func registerUserRoutes(mux *http.ServeMux, handler *UserPageHandler, authentication *middleware.AuthenticationMiddleware, csrf *middleware.CSRFMiddleware) {
	mux.Handle("GET /users/student-options", authentication.RequireAuthentication(http.HandlerFunc(handler.StudentOptions)))
	mux.Handle("GET /users", authentication.RequireAuthentication(csrf.Protect(http.HandlerFunc(handler.List))))
	mux.Handle("POST /users", authentication.RequireAuthentication(limitAcademicYearRequestBody(csrf.Protect(http.HandlerFunc(handler.Create)))))
	mux.Handle("POST /users/{id}/reset-password", authentication.RequireAuthentication(limitAcademicYearRequestBody(csrf.Protect(http.HandlerFunc(handler.ResetPassword)))))
	mux.Handle("GET /account/password", authentication.RequireAuthentication(csrf.Protect(http.HandlerFunc(handler.ChangePassword))))
	mux.Handle("POST /account/password", authentication.RequireAuthentication(limitAcademicYearRequestBody(csrf.Protect(http.HandlerFunc(handler.ChangePassword)))))
}

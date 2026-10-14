package http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/google/uuid"
)

type teacherAssignmentRouteDependencies struct {
	years   *academicYearServiceFake
	classes *classServiceFake
	users   *teacherAssignmentUserServiceFake
}

func newTeacherAssignmentRouterWithSession(
	t *testing.T,
	role domain.UserRole,
	assignments *teacherAssignmentServiceFake,
) (http.Handler, string, string) {
	router, sessionToken, csrfToken, _ := newTeacherAssignmentRouterWithDependencies(t, role, assignments)
	return router, sessionToken, csrfToken
}

func newTeacherAssignmentRouterWithDependencies(
	t *testing.T,
	role domain.UserRole,
	assignments *teacherAssignmentServiceFake,
) (http.Handler, string, string, teacherAssignmentRouteDependencies) {
	t.Helper()

	userID := uuid.New()
	sessionToken := "teacher-assignment-session-" + uuid.NewString()
	session := domain.Session{
		ID: uuid.New(), UserID: userID,
		TokenHash: security.HashSessionToken(sessionToken),
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
	}
	user := domain.User{ID: userID, Username: "route-test-user", Role: role}
	auth, err := middleware.NewAuthenticationMiddleware(
		newTestSessionCookie(t),
		&routerSessionRepositoryFake{session: session},
		&routerUserRepositoryFake{user: user},
	)
	if err != nil {
		t.Fatal(err)
	}

	templates := newTeacherAssignmentTestTemplate(t)
	years := &academicYearServiceFake{}
	classes := &classServiceFake{}
	users := &teacherAssignmentUserServiceFake{}
	assignmentHandler, err := NewTeacherAssignmentPageHandler(assignments, users, classes, years, templates)
	if err != nil {
		t.Fatal(err)
	}
	academicYearHandler, err := NewAcademicYearPageHandler(years, templates)
	if err != nil {
		t.Fatal(err)
	}
	classHandler, err := NewClassPageHandler(classes, years, templates)
	if err != nil {
		t.Fatal(err)
	}
	dashboard, err := NewDashboardPageHandler(templates)
	if err != nil {
		t.Fatal(err)
	}

	router, err := NewRouter(
		HandlerSet{
			Authentication:    &AuthenticationHandler{},
			LoginPage:         newTestLoginPageHandler(t),
			Dashboard:         dashboard,
			AcademicYear:      academicYearHandler,
			Class:             classHandler,
			TeacherAssignment: assignmentHandler,
		},
		MiddlewareSet{
			Authentication: auth,
			CSRF:           middleware.NewCSRFMiddleware(newTestCSRFCookie(t)),
		},
		newTestStaticAssetHandler(),
	)
	if err != nil {
		t.Fatal(err)
	}

	csrfToken, err := security.GenerateCSRFToken()
	if err != nil {
		t.Fatal(err)
	}
	return router, sessionToken, csrfToken, teacherAssignmentRouteDependencies{years: years, classes: classes, users: users}
}

func TestTeacherAssignmentRoutesRequireAuthentication(t *testing.T) {
	assignments := &teacherAssignmentServiceFake{}
	router, _, _ := newTeacherAssignmentRouterWithSession(t, domain.RoleAdmin, assignments)

	for _, tc := range []struct{ method, target string }{
		{http.MethodGet, "/teacher-assignments"},
		{http.MethodPost, "/teacher-assignments"},
	} {
		t.Run(tc.method, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, nil)
			if tc.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s status = %d, want %d; body=%s", tc.method, tc.target, w.Code, http.StatusUnauthorized, w.Body.String())
			}
		})
	}
	if assignments.assignCalled {
		t.Fatal("assignment service must not be called for unauthenticated requests")
	}
}

func TestTeacherAssignmentPostRequiresValidCSRF(t *testing.T) {
	assignments := &teacherAssignmentServiceFake{}
	router, sessionToken, _ := newTeacherAssignmentRouterWithSession(t, domain.RoleAdmin, assignments)
	form := url.Values{
		"teacher_id":       {uuid.NewString()},
		"class_id":         {uuid.NewString()},
		"academic_year_id": {uuid.NewString()},
	}
	req := httptest.NewRequest(http.MethodPost, "/teacher-assignments", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: security.DefaultSessionCookieName, Value: sessionToken})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("POST without CSRF status = %d, want %d; body=%s", w.Code, http.StatusForbidden, w.Body.String())
	}
	if assignments.assignCalled {
		t.Fatal("assignment service must not be called when CSRF validation fails")
	}
}

func TestTeacherAssignmentGetDeniesNonAdminBeforeLoadingData(t *testing.T) {
	assignments := &teacherAssignmentServiceFake{}
	router, sessionToken, _, deps := newTeacherAssignmentRouterWithDependencies(t, domain.RoleWaliKelas, assignments)
	req := httptest.NewRequest(http.MethodGet, "/teacher-assignments", nil)
	req.AddCookie(&http.Cookie{Name: security.DefaultSessionCookieName, Value: sessionToken})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("GET /teacher-assignments status = %d, want %d; body=%s", w.Code, http.StatusForbidden, w.Body.String())
	}
	if deps.years.listCalled || deps.users.listCalled || deps.classes.listCalled {
		t.Fatalf("non-admin request loaded protected data: years.listCalled=%v users.listCalled=%v classes.listCalled=%v", deps.years.listCalled, deps.users.listCalled, deps.classes.listCalled)
	}
	if assignments.assignCalled {
		t.Fatal("assignment service must not be called for non-admin GET")
	}
}

package http

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type routeAuthorizationClassService struct {
	class       *domain.Class
	listCalls   int
	createCalls int
	getCalls    int
	updateCalls int
	lastList    service.ListClassesInput
	lastCreate  service.CreateClassInput
	lastGet     service.GetClassInput
	lastUpdate  service.UpdateClassInput
}

func (f *routeAuthorizationClassService) Create(_ context.Context, input service.CreateClassInput) (service.CreateClassOutput, error) {
	f.createCalls++
	f.lastCreate = input
	if input.Actor.Role != domain.RoleAdmin {
		return service.CreateClassOutput{}, service.ErrForbidden
	}
	class, err := domain.NewClass(input.Name, input.Level)
	if err != nil {
		return service.CreateClassOutput{}, err
	}
	class.ID = uuid.New()
	f.class = &class
	return service.CreateClassOutput{Class: &class}, nil
}

func (f *routeAuthorizationClassService) Get(_ context.Context, input service.GetClassInput) (service.GetClassOutput, error) {
	f.getCalls++
	f.lastGet = input
	if input.Actor.Role != domain.RoleAdmin {
		return service.GetClassOutput{}, service.ErrForbidden
	}
	if f.class == nil || f.class.ID != input.ClassID {
		return service.GetClassOutput{}, repository.ErrNotFound
	}
	return service.GetClassOutput{Class: f.class}, nil
}

func (f *routeAuthorizationClassService) List(_ context.Context, input service.ListClassesInput) (service.ListClassesOutput, error) {
	f.listCalls++
	f.lastList = input
	if f.class == nil {
		return service.ListClassesOutput{}, nil
	}
	return service.ListClassesOutput{Classes: []*domain.Class{f.class}}, nil
}

func (f *routeAuthorizationClassService) Update(_ context.Context, input service.UpdateClassInput) (service.UpdateClassOutput, error) {
	f.updateCalls++
	f.lastUpdate = input
	if input.Actor.Role != domain.RoleAdmin {
		return service.UpdateClassOutput{}, service.ErrForbidden
	}
	class, err := domain.NewClass(input.Name, input.Level)
	if err != nil {
		return service.UpdateClassOutput{}, err
	}
	class.ID = input.ClassID
	f.class = &class
	return service.UpdateClassOutput{Class: &class}, nil
}

func newClassRouterWithSession(
	t *testing.T,
	role domain.UserRole,
	username string,
	classes *routeAuthorizationClassService,
) (http.Handler, string, string) {
	t.Helper()

	userID := uuid.New()
	sessionToken := "class-route-session-" + uuid.NewString()
	session := domain.Session{
		ID: uuid.New(), UserID: userID,
		TokenHash: security.HashSessionToken(sessionToken),
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
	}
	user := domain.User{ID: userID, Username: username, Role: role}

	auth, err := middleware.NewAuthenticationMiddleware(
		newTestSessionCookie(t),
		&routerSessionRepositoryFake{session: session},
		&routerUserRepositoryFake{user: user},
	)
	if err != nil {
		t.Fatal(err)
	}

	templates, err := template.New("pages").Parse(
		`{{define "auth"}}<html><body>{{.Title}}</body></html>{{end}}` +
			`{{define "base"}}<html><body>{{.Title}}</body></html>{{end}}` +
			`{{define "academic-years"}}{{.Username}}|{{.Role}}|{{.Heading}}{{end}}` +
			`{{define "classes"}}{{.Username}}|{{.Role}}|{{.Heading}}{{range .AcademicYears}}{{.Name}}{{end}}{{range .Rows}}{{.Name}}{{end}}{{if .Error}}{{.Error}}{{end}}{{end}}`,
	)
	if err != nil {
		t.Fatal(err)
	}

	classHandler, err := NewClassPageHandler(
		classes,
		&academicYearServiceFake{years: []*domain.AcademicYear{
			{ID: uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"), Name: "2026/2027"},
		}},
		templates,
	)
	if err != nil {
		t.Fatal(err)
	}
	academicYearHandler, err := NewAcademicYearPageHandler(&academicYearServiceFake{}, templates)
	if err != nil {
		t.Fatal(err)
	}
	dashboard, err := NewDashboardPageHandler(templates)
	if err != nil {
		t.Fatal(err)
	}

	router, err := NewRouter(
		HandlerSet{
			Authentication: &AuthenticationHandler{},
			LoginPage:      newTestLoginPageHandler(t),
			Dashboard:      dashboard,
			AcademicYear:   academicYearHandler,
			Class:          classHandler,
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
	return router, sessionToken, csrfToken
}

func classRouteRequest(
	router http.Handler,
	method, target, sessionToken, csrfToken string,
	form url.Values,
	includeCSRFCookie bool,
) *httptest.ResponseRecorder {
	var body strings.Reader
	if form != nil {
		form = cloneValues(form)
		if csrfToken != "" {
			form.Set("csrf_token", csrfToken)
		}
		body = *strings.NewReader(form.Encode())
	}
	var req *http.Request
	if form != nil {
		req = httptest.NewRequest(method, target, &body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	if sessionToken != "" {
		req.AddCookie(&http.Cookie{Name: security.DefaultSessionCookieName, Value: sessionToken})
	}
	if includeCSRFCookie && csrfToken != "" {
		req.AddCookie(&http.Cookie{Name: security.DefaultCSRFCookieName, Value: csrfToken})
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func cloneValues(values url.Values) url.Values {
	cloned := make(url.Values, len(values))
	for key, entries := range values {
		cloned[key] = append([]string(nil), entries...)
	}
	return cloned
}

func TestClassRoutesAllowAuthenticatedRolesToReadList(t *testing.T) {
	yearID := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	for _, role := range []domain.UserRole{domain.RoleAdmin, domain.RoleWaliKelas, domain.RoleSiswa} {
		t.Run(string(role), func(t *testing.T) {
			class := &domain.Class{ID: uuid.New(), Name: "VII A", Level: 7}
			fake := &routeAuthorizationClassService{class: class}
			router, session, _ := newClassRouterWithSession(t, role, "class-list-"+string(role), fake)
			recorder := classRouteRequest(router, http.MethodGet, "/classes?academic_year_id="+yearID.String(), session, "", nil, false)
			if recorder.Code != http.StatusOK {
				t.Fatalf("GET /classes status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
			}
			if fake.listCalls != 1 || fake.lastList.Actor.Role != role {
				t.Fatalf("List calls/actor = %d/%+v, want one call for role %s", fake.listCalls, fake.lastList.Actor, role)
			}
		})
	}
}

func TestClassRoutesDenyNonAdminWritesAndDetail(t *testing.T) {
	classID := uuid.New()
	for _, role := range []domain.UserRole{domain.RoleWaliKelas, domain.RoleSiswa} {
		t.Run(string(role), func(t *testing.T) {
			fake := &routeAuthorizationClassService{class: &domain.Class{ID: classID, Name: "VII A", Level: 7}}
			router, session, csrf := newClassRouterWithSession(t, role, "class-denied-"+string(role), fake)

			createForm := url.Values{"name": {"VII B"}, "level": {"7"}}
			createResponse := classRouteRequest(router, http.MethodPost, "/classes", session, csrf, createForm, true)
			if createResponse.Code != http.StatusForbidden {
				t.Fatalf("POST /classes status = %d, want 403; body=%s", createResponse.Code, createResponse.Body.String())
			}
			if fake.createCalls != 1 || fake.lastCreate.Actor.Role != role {
				t.Fatalf("Create calls/actor = %d/%+v, want service denial for %s", fake.createCalls, fake.lastCreate.Actor, role)
			}

			detailResponse := classRouteRequest(router, http.MethodGet, "/classes/"+classID.String(), session, "", nil, false)
			if detailResponse.Code != http.StatusForbidden {
				t.Fatalf("GET /classes/{id} status = %d, want 403; body=%s", detailResponse.Code, detailResponse.Body.String())
			}
			if fake.getCalls != 1 || fake.lastGet.Actor.Role != role {
				t.Fatalf("Get calls/actor = %d/%+v, want service denial for %s", fake.getCalls, fake.lastGet.Actor, role)
			}

			updateForm := url.Values{"name": {"VII B"}, "level": {"7"}}
			updateResponse := classRouteRequest(router, http.MethodPost, "/classes/"+classID.String(), session, csrf, updateForm, true)
			if updateResponse.Code != http.StatusForbidden {
				t.Fatalf("POST /classes/{id} status = %d, want 403; body=%s", updateResponse.Code, updateResponse.Body.String())
			}
			if fake.updateCalls != 1 || fake.lastUpdate.Actor.Role != role {
				t.Fatalf("Update calls/actor = %d/%+v, want service denial for %s", fake.updateCalls, fake.lastUpdate.Actor, role)
			}
		})
	}
}

func TestClassRoutesAllowAdminCreate(t *testing.T) {
	fake := &routeAuthorizationClassService{}
	router, session, csrf := newClassRouterWithSession(t, domain.RoleAdmin, "class-admin-create", fake)
	form := url.Values{"name": {"VII A"}, "level": {"7"}}
	recorder := classRouteRequest(router, http.MethodPost, "/classes", session, csrf, form, true)
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("POST /classes status = %d, want 303; body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Location") != "/classes" {
		t.Fatalf("Location = %q, want /classes", recorder.Header().Get("Location"))
	}
	if fake.createCalls != 1 || fake.lastCreate.Actor.Role != domain.RoleAdmin {
		t.Fatalf("Create calls/actor = %d/%+v, want one Admin call", fake.createCalls, fake.lastCreate.Actor)
	}
}

func TestClassRoutesRejectPostWithoutValidCSRFBeforeService(t *testing.T) {
	fake := &routeAuthorizationClassService{}
	router, session, _ := newClassRouterWithSession(t, domain.RoleAdmin, "class-csrf-test", fake)
	form := url.Values{"name": {"VII A"}, "level": {"7"}}
	recorder := classRouteRequest(router, http.MethodPost, "/classes", session, "", form, false)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("POST without CSRF cookie status = %d, want 403; body=%s", recorder.Code, recorder.Body.String())
	}
	if fake.createCalls != 0 {
		t.Fatalf("Create calls = %d, want 0 when CSRF validation fails", fake.createCalls)
	}
}

func TestClassRoutesRejectUnauthenticatedRequestsBeforeService(t *testing.T) {
	fake := &routeAuthorizationClassService{}
	router, _, csrf := newClassRouterWithSession(t, domain.RoleAdmin, "class-no-session", fake)
	recorder := classRouteRequest(router, http.MethodGet, "/classes", "", csrf, nil, false)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("GET /classes without session status = %d, want 401; body=%s", recorder.Code, recorder.Body.String())
	}
	if fake.listCalls != 0 || fake.createCalls != 0 || fake.getCalls != 0 || fake.updateCalls != 0 {
		t.Fatalf("service calls without session = list:%d create:%d get:%d update:%d; want all zero",
			fake.listCalls, fake.createCalls, fake.getCalls, fake.updateCalls)
	}
}

var _ service.ClassService = (*routeAuthorizationClassService)(nil)

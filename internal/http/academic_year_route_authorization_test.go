package http

import (
	"context"
	"errors"
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

// routeAuthorizationAcademicYearService enforces the same role boundary as the
// application service while recording the actor that reached each operation.
// The route tests therefore cover session authentication, route matching, CSRF,
// actor propagation, and HTTP error mapping without requiring PostgreSQL.
type routeAuthorizationAcademicYearService struct {
	year                *domain.AcademicYear
	lastListActor       service.Actor
	lastGetActor        service.Actor
	lastTransition      string
	lastTransitionActor service.Actor
}

func (f *routeAuthorizationAcademicYearService) Create(context.Context, service.CreateAcademicYearInput) (service.CreateAcademicYearOutput, error) {
	return service.CreateAcademicYearOutput{}, errors.New("unexpected Create call")
}

func (f *routeAuthorizationAcademicYearService) Get(_ context.Context, input service.GetAcademicYearInput) (service.GetAcademicYearOutput, error) {
	f.lastGetActor = input.Actor
	if f.year == nil {
		return service.GetAcademicYearOutput{}, repository.ErrNotFound
	}
	return service.GetAcademicYearOutput{AcademicYear: f.year}, nil
}

func (f *routeAuthorizationAcademicYearService) List(_ context.Context, input service.ListAcademicYearsInput) (service.ListAcademicYearsOutput, error) {
	f.lastListActor = input.Actor
	var years []*domain.AcademicYear
	if f.year != nil {
		years = append(years, f.year)
	}
	return service.ListAcademicYearsOutput{AcademicYears: years}, nil
}

func (f *routeAuthorizationAcademicYearService) StartClosing(_ context.Context, input service.StartClosingAcademicYearInput) (service.StartClosingAcademicYearOutput, error) {
	f.lastTransition, f.lastTransitionActor = "start-closing", input.Actor
	if input.Actor.Role != domain.RoleAdmin {
		return service.StartClosingAcademicYearOutput{}, service.ErrForbidden
	}
	return service.StartClosingAcademicYearOutput{AcademicYear: f.year}, nil
}

func (f *routeAuthorizationAcademicYearService) Close(_ context.Context, input service.CloseAcademicYearInput) (service.CloseAcademicYearOutput, error) {
	f.lastTransition, f.lastTransitionActor = "close", input.Actor
	if input.Actor.Role != domain.RoleAdmin {
		return service.CloseAcademicYearOutput{}, service.ErrForbidden
	}
	return service.CloseAcademicYearOutput{AcademicYear: f.year}, nil
}

func (f *routeAuthorizationAcademicYearService) Reopen(_ context.Context, input service.ReopenAcademicYearInput) (service.ReopenAcademicYearOutput, error) {
	f.lastTransition, f.lastTransitionActor = "reopen", input.Actor
	if input.Actor.Role != domain.RoleAdmin {
		return service.ReopenAcademicYearOutput{}, service.ErrForbidden
	}
	return service.ReopenAcademicYearOutput{AcademicYear: f.year}, nil
}

func newAcademicYearRouterWithSession(
	t *testing.T,
	role domain.UserRole,
	username string,
	academicYears *routeAuthorizationAcademicYearService,
) (http.Handler, string, string) {
	t.Helper()

	userID := uuid.New()
	sessionToken := "academic-year-route-session-" + uuid.NewString()
	session := domain.Session{
		ID: uuid.New(), UserID: userID,
		TokenHash: security.HashSessionToken(sessionToken),
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
	}
	user := domain.User{ID: userID, Username: username, Role: role}

	auth, err := middleware.NewAuthenticationMiddleware(
		security.NewDefaultSessionCookie(),
		&routerSessionRepositoryFake{session: session},
		&routerUserRepositoryFake{user: user},
	)
	if err != nil {
		t.Fatal(err)
	}
	templates := template.Must(template.New("pages").Parse(
		`{{define "auth"}}<html><body>{{.Title}}</body></html>{{end}}` +
			`{{define "base"}}<html><body>{{.Title}}</body></html>{{end}}` +
			`{{define "academic-years"}}{{.Username}}|{{.Role}}|{{.Heading}}{{end}}`,
	))
	academicYearHandler, err := NewAcademicYearPageHandler(
		academicYears,
		templates,
	)
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
			Class:          newTestClassPageHandler(t),
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

func TestAcademicYearRoutesAllowAuthenticatedWaliKelasToReadListAndDetail(t *testing.T) {
	year := academicYearsForTest(1)[0]
	fake := &routeAuthorizationAcademicYearService{year: year}
	router, sessionToken, _ := newAcademicYearRouterWithSession(
		t, domain.RoleWaliKelas, "wali-kelas-route-test", fake,
	)

	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "list", path: "/academic-years"},
		{name: "detail", path: "/academic-years/" + year.ID.String()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.AddCookie(&http.Cookie{
				Name: security.DefaultSessionCookieName, Value: sessionToken,
			})
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("GET %s status = %d, want 200; body=%s", tc.path, recorder.Code, recorder.Body.String())
			}
			if len(recorder.Result().Cookies()) == 0 {
				t.Fatal("authenticated GET did not establish a CSRF cookie")
			}
		})
	}

	if fake.lastListActor.Role != domain.RoleWaliKelas || fake.lastListActor.Username != "wali-kelas-route-test" {
		t.Fatalf("list actor = %+v, want authenticated Wali Kelas actor", fake.lastListActor)
	}
	if fake.lastGetActor.Role != domain.RoleWaliKelas || fake.lastGetActor.Username != "wali-kelas-route-test" {
		t.Fatalf("detail actor = %+v, want authenticated Wali Kelas actor", fake.lastGetActor)
	}
}

func TestAcademicYearTransitionRoutesDenyWaliKelasAndAllowAdmin(t *testing.T) {
	for _, action := range []string{"start-closing", "close", "reopen"} {
		t.Run(action, func(t *testing.T) {
			year := academicYearsForTest(1)[0]
			fake := &routeAuthorizationAcademicYearService{year: year}
			waliRouter, waliSession, csrfToken := newAcademicYearRouterWithSession(
				t, domain.RoleWaliKelas, "wali-kelas-transition-test", fake,
			)
			postTransition := func(router http.Handler, sessionToken string) *httptest.ResponseRecorder {
				form := url.Values{"csrf_token": {csrfToken}}
				req := httptest.NewRequest(
					http.MethodPost,
					"/academic-years/"+year.ID.String()+"/"+action,
					strings.NewReader(form.Encode()),
				)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.AddCookie(&http.Cookie{Name: security.DefaultSessionCookieName, Value: sessionToken})
				req.AddCookie(&http.Cookie{Name: security.DefaultCSRFCookieName, Value: csrfToken})
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, req)
				return recorder
			}

			waliResponse := postTransition(waliRouter, waliSession)
			if waliResponse.Code != http.StatusForbidden {
				t.Fatalf("Wali Kelas POST %s status = %d, want 403; body=%s", action, waliResponse.Code, waliResponse.Body.String())
			}
			if fake.lastTransition != action || fake.lastTransitionActor.Role != domain.RoleWaliKelas {
				t.Fatalf("transition reached service as %+v, want %s with Wali Kelas actor", fake.lastTransitionActor, action)
			}

			adminRouter, adminSession, adminCSRF := newAcademicYearRouterWithSession(
				t, domain.RoleAdmin, "admin-transition-test", fake,
			)
			// Each authenticated identity receives its own CSRF token.
			csrfToken = adminCSRF
			adminResponse := postTransition(adminRouter, adminSession)
			if adminResponse.Code != http.StatusSeeOther {
				t.Fatalf("Admin POST %s status = %d, want 303; body=%s", action, adminResponse.Code, adminResponse.Body.String())
			}
			if got := adminResponse.Header().Get("Location"); got != "/academic-years/"+year.ID.String() {
				t.Fatalf("Admin redirect = %q, want detail URL", got)
			}
			if fake.lastTransition != action || fake.lastTransitionActor.Role != domain.RoleAdmin {
				t.Fatalf("transition reached service as %+v, want Admin actor", fake.lastTransitionActor)
			}
		})
	}
}

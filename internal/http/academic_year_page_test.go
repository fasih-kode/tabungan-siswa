package http

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func newAcademicYearHandlerForTest(t *testing.T, fake *academicYearServiceFake) *AcademicYearPageHandler {
	t.Helper()
	tmpl := template.Must(template.New("academic-years").Parse(`{{define "academic-years"}}{{.Heading}}|{{.Error}}{{end}}`))
	h, err := NewAcademicYearPageHandler(fake, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func requestWithActor(method, target string, actor service.Actor) *http.Request {
	r := httptest.NewRequest(method, target, nil)
	return r.WithContext(service.WithActor(r.Context(), actor))
}

func TestWaliKelasCanRenderAcademicYearListAndDetailWithHeaderContext(t *testing.T) {
	year := academicYearsForTest(1)[0]
	actor := service.Actor{
		UserID:   uuid.New(),
		Username: "wali-kelas-test",
		Role:     domain.RoleWaliKelas,
	}

	tmpl := template.Must(template.New("academic-years").Parse(
		`{{define "academic-years"}}{{.Username}}|{{.Role}}|{{.CSRFToken}}|{{.Heading}}{{end}}`,
	))
	fake := &academicYearServiceFake{years: []*domain.AcademicYear{year}}
	handler, err := NewAcademicYearPageHandler(fake, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	csrf := middleware.NewCSRFMiddleware(newTestCSRFCookie(t))

	t.Run("list", func(t *testing.T) {
		req := requestWithActor(http.MethodGet, "/academic-years", actor)
		recorder := httptest.NewRecorder()
		csrf.Protect(http.HandlerFunc(handler.List)).ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
		}
		body := recorder.Body.String()
		for _, want := range []string{"wali-kelas-test", "WALI_KELAS", "Tahun Ajaran"} {
			if !strings.Contains(body, want) {
				t.Errorf("list body %q does not contain %q", body, want)
			}
		}
		if strings.Count(body, "|") != 3 {
			t.Errorf("list body %q does not include username, role, CSRF token and heading", body)
		}
	})

	t.Run("detail", func(t *testing.T) {
		req := requestWithActor(http.MethodGet, "/academic-years/"+year.ID.String(), actor)
		req.SetPathValue("id", year.ID.String())
		recorder := httptest.NewRecorder()
		csrf.Protect(http.HandlerFunc(handler.Detail)).ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
		}
		body := recorder.Body.String()
		for _, want := range []string{"wali-kelas-test", "WALI_KELAS", year.Name} {
			if !strings.Contains(body, want) {
				t.Errorf("detail body %q does not contain %q", body, want)
			}
		}
		if strings.Count(body, "|") != 3 {
			t.Errorf("detail body %q does not include username, role, CSRF token and heading", body)
		}
	})
}

func TestAcademicYearListUsesBoundedPagination(t *testing.T) {
	fake := &academicYearServiceFake{total: 30}
	handler := newAcademicYearHandlerForTest(t, fake)
	csrf := middleware.NewCSRFMiddleware(newTestCSRFCookie(t))
	actor := service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}
	req := requestWithActor(http.MethodGet, "/academic-years?page=3&limit=25", actor)
	recorder := httptest.NewRecorder()
	csrf.Protect(http.HandlerFunc(handler.List)).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if fake.lastListInput.Options.Limit != 11 || fake.lastListInput.Options.Offset != 20 {
		t.Fatalf("pagination = %+v, want limit 11 (10 visible + 1 lookahead) offset 20", fake.lastListInput.Options)
	}
	if fake.lastListInput.Actor != actor {
		t.Fatalf("actor = %+v, want %+v", fake.lastListInput.Actor, actor)
	}
}

func TestAcademicYearListRejectsInvalidPagination(t *testing.T) {
	for _, target := range []string{"/academic-years?page=0", "/academic-years?page=x", "/academic-years?limit=101", "/academic-years?limit=0"} {
		t.Run(target, func(t *testing.T) {
			handler := newAcademicYearHandlerForTest(t, &academicYearServiceFake{})
			req := requestWithActor(http.MethodGet, target, service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin})
			recorder := httptest.NewRecorder()
			middleware.NewCSRFMiddleware(newTestCSRFCookie(t)).Protect(http.HandlerFunc(handler.List)).ServeHTTP(recorder, req)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", recorder.Code)
			}
		})
	}
}

func TestAcademicYearCreateRedirectsAfterSuccess(t *testing.T) {
	fake := &academicYearServiceFake{}
	handler := newAcademicYearHandlerForTest(t, fake)
	form := url.Values{"csrf_token": {"token"}, "name": {"2026/2027"}, "start_date": {"2026-07-01"}, "end_date": {"2027-06-30"}}
	req := httptest.NewRequest(http.MethodPost, "/academic-years", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}))
	recorder := httptest.NewRecorder()
	handler.Create(recorder, req)
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Location") != "/academic-years" {
		t.Fatalf("Location = %q", recorder.Header().Get("Location"))
	}
}

func TestAcademicYearCreateRejectsInvalidDateRange(t *testing.T) {
	handler := newAcademicYearHandlerForTest(t, &academicYearServiceFake{})
	form := url.Values{"csrf_token": {"token"}, "name": {"2026/2027"}, "start_date": {"2027-06-30"}, "end_date": {"2026-07-01"}}
	req := httptest.NewRequest(http.MethodPost, "/academic-years", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}))
	recorder := httptest.NewRecorder()
	handler.Create(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "Tanggal mulai harus lebih awal") {
		t.Fatalf("body does not contain safe validation message: %s", recorder.Body.String())
	}
}

func TestAcademicYearCreateConflictRendersExistingYears(t *testing.T) {
	existing := academicYearsForTest(1)[0]
	fake := &academicYearServiceFake{
		years:     []*domain.AcademicYear{existing},
		createErr: repository.ErrConflict,
	}
	tmpl := template.Must(template.New("academic-years").Parse(
		`{{define "academic-years"}}{{.Error}}|{{range .Rows}}{{.Name}} {{.Status}}{{end}}{{end}}`,
	))
	handler, err := NewAcademicYearPageHandler(fake, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"csrf_token": {"token"}, "name": {"2026/2027"}, "start_date": {"2026-07-01"}, "end_date": {"2027-06-30"}}
	req := httptest.NewRequest(http.MethodPost, "/academic-years", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}))
	recorder := httptest.NewRecorder()
	handler.Create(recorder, req)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", recorder.Code, recorder.Body.String())
	}
	for _, want := range []string{"Perubahan tidak dapat dilakukan", existing.Name, string(existing.Status)} {
		if !strings.Contains(recorder.Body.String(), want) {
			t.Errorf("body %q does not contain %q", recorder.Body.String(), want)
		}
	}
	if !fake.listCalled {
		t.Fatal("existing academic years were not reloaded after create conflict")
	}
	if fake.lastListInput.Options.Limit != defaultAcademicYearPageSize+1 || !fake.lastListInput.Options.IncludeTotal {
		t.Fatalf("list options = %+v, want first page with lookahead and total", fake.lastListInput.Options)
	}
}

func TestAcademicYearTransitionRedirectsToDetail(t *testing.T) {
	fake := &academicYearServiceFake{}
	handler := newAcademicYearHandlerForTest(t, fake)
	id := uuid.New()
	form := url.Values{"csrf_token": {"token"}}
	req := httptest.NewRequest(http.MethodPost, "/academic-years/"+id.String()+"/start-closing", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", id.String())
	req.SetPathValue("action", "start-closing")
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}))
	recorder := httptest.NewRecorder()
	handler.Transition(recorder, req)
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Location") != "/academic-years/"+id.String() {
		t.Fatalf("Location = %q", recorder.Header().Get("Location"))
	}
	if fake.lastTransition != "start-closing" {
		t.Fatalf("transition = %q, want start-closing", fake.lastTransition)
	}
}

func TestAcademicYearCloseFailureExplainsUnsettledAccounts(t *testing.T) {
	year := *academicYearsForTest(1)[0]
	year.Status = domain.AcademicYearClosing
	fake := &academicYearServiceFake{years: []*domain.AcademicYear{&year}, transitionErr: service.ErrAcademicYearHasUnsettledAccounts}
	handler := newAcademicYearHandlerForTest(t, fake)
	form := url.Values{"csrf_token": {"token"}}
	req := httptest.NewRequest(http.MethodPost, "/academic-years/"+year.ID.String()+"/close", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", year.ID.String())
	req.SetPathValue("action", "close")
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}))
	recorder := httptest.NewRecorder()
	handler.Transition(recorder, req)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", recorder.Code, recorder.Body.String())
	}
	for _, want := range []string{"rekening tabungan yang belum diselesaikan", "Selesaikan settlement", year.Name} {
		if !strings.Contains(recorder.Body.String(), want) {
			t.Errorf("body %q does not contain %q", recorder.Body.String(), want)
		}
	}
	if strings.TrimSpace(recorder.Body.String()) == http.StatusText(http.StatusConflict) {
		t.Fatal("transition failure rendered only the raw status text")
	}
}

func TestAcademicYearListUsesLookaheadForNextPage(t *testing.T) {
	actor := service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}
	for _, tc := range []struct {
		name        string
		count       int
		wantHasNext bool
		wantRows    int
	}{
		{name: "more rows than page", count: 11, wantHasNext: true, wantRows: 10},
		{name: "exact page boundary", count: 10, wantHasNext: false, wantRows: 10},
		{name: "last partial page", count: 7, wantHasNext: false, wantRows: 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &academicYearServiceFake{years: academicYearsForTest(tc.count)}
			tmpl := template.Must(template.New("academic-years").Parse(`{{define "academic-years"}}{{.HasNext}}|{{len .Rows}}{{end}}`))
			handler, err := NewAcademicYearPageHandler(fake, tmpl)
			if err != nil {
				t.Fatal(err)
			}
			req := requestWithActor(http.MethodGet, "/academic-years?limit=20", actor)
			recorder := httptest.NewRecorder()
			middleware.NewCSRFMiddleware(newTestCSRFCookie(t)).Protect(http.HandlerFunc(handler.List)).ServeHTTP(recorder, req)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
			}
			want := strings.Join([]string{strconv.FormatBool(tc.wantHasNext), strconv.Itoa(tc.wantRows)}, "|")
			if recorder.Body.String() != want {
				t.Fatalf("rendered view model = %q, want %q", recorder.Body.String(), want)
			}
			if fake.lastListInput.Options.Limit != 11 {
				t.Fatalf("service limit = %d, want 11 (10 visible + 1 lookahead)", fake.lastListInput.Options.Limit)
			}
		})
	}
}

func academicYearsForTest(count int) []*domain.AcademicYear {
	years := make([]*domain.AcademicYear, 0, count)
	for i := 0; i < count; i++ {
		year, err := domain.NewAcademicYear(
			"2026/2027",
			time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2027, time.June, 30, 0, 0, 0, 0, time.UTC),
		)
		if err != nil {
			panic(err)
		}
		years = append(years, &year)
	}
	return years
}

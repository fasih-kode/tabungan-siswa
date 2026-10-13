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

func TestAcademicYearListUsesBoundedPagination(t *testing.T) {
	fake := &academicYearServiceFake{}
	handler := newAcademicYearHandlerForTest(t, fake)
	csrf := middleware.NewCSRFMiddleware(newTestCSRFCookie(t))
	actor := service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}
	req := requestWithActor(http.MethodGet, "/academic-years?page=3&limit=25", actor)
	recorder := httptest.NewRecorder()
	csrf.Protect(http.HandlerFunc(handler.List)).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if fake.lastListInput.Options.Limit != 26 || fake.lastListInput.Options.Offset != 50 {
		t.Fatalf("pagination = %+v, want limit 26 (25 visible + 1 lookahead) offset 50", fake.lastListInput.Options)
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

func TestAcademicYearListUsesLookaheadForNextPage(t *testing.T) {
	actor := service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}
	for _, tc := range []struct {
		name        string
		count       int
		wantHasNext bool
		wantRows    int
	}{
		{name: "more rows than page", count: 21, wantHasNext: true, wantRows: 20},
		{name: "exact page boundary", count: 20, wantHasNext: false, wantRows: 20},
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
			if fake.lastListInput.Options.Limit != 21 {
				t.Fatalf("service limit = %d, want 21 (20 visible + 1 lookahead)", fake.lastListInput.Options.Limit)
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

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

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type classServiceFake struct {
	classes                               []*domain.Class
	created                               service.CreateClassInput
	updated                               service.UpdateClassInput
	createErr, updateErr, getErr, listErr error
	listInput                             service.ListClassesInput
}

func (f *classServiceFake) Create(_ context.Context, in service.CreateClassInput) (service.CreateClassOutput, error) {
	f.created = in
	if f.createErr != nil {
		return service.CreateClassOutput{}, f.createErr
	}
	c, _ := domain.NewClass(in.Name, in.Level)
	f.classes = append(f.classes, &c)
	return service.CreateClassOutput{Class: &c}, nil
}
func (f *classServiceFake) Get(_ context.Context, in service.GetClassInput) (service.GetClassOutput, error) {
	if f.getErr != nil {
		return service.GetClassOutput{}, f.getErr
	}
	for _, c := range f.classes {
		if c.ID == in.ClassID {
			return service.GetClassOutput{Class: c}, nil
		}
	}
	return service.GetClassOutput{}, errors.New("not found")
}
func (f *classServiceFake) List(_ context.Context, in service.ListClassesInput) (service.ListClassesOutput, error) {
	f.listInput = in
	if f.listErr != nil {
		return service.ListClassesOutput{}, f.listErr
	}
	start := in.Options.Offset
	if start < 0 || start >= len(f.classes) {
		return service.ListClassesOutput{Classes: []*domain.Class{}}, nil
	}
	end := start + in.Options.Limit
	if in.Options.Limit <= 0 || end > len(f.classes) {
		end = len(f.classes)
	}
	return service.ListClassesOutput{Classes: f.classes[start:end]}, nil
}
func (f *classServiceFake) Update(_ context.Context, in service.UpdateClassInput) (service.UpdateClassOutput, error) {
	f.updated = in
	if f.updateErr != nil {
		return service.UpdateClassOutput{}, f.updateErr
	}
	c, err := domain.NewClass(in.Name, in.Level)
	if err != nil {
		return service.UpdateClassOutput{}, err
	}
	c.ID = in.ClassID
	return service.UpdateClassOutput{Class: &c}, nil
}

func TestClassPageListRendersScopedClassesAndPagination(t *testing.T) {
	yearID := uuid.New()
	classID := uuid.New()
	year := &domain.AcademicYear{ID: yearID, Name: "2026/2027"}
	class := &domain.Class{ID: classID, Name: "VII A", Level: 7}
	years := &academicYearServiceFake{years: []*domain.AcademicYear{year}}
	classes := &classServiceFake{classes: []*domain.Class{class}}
	h, err := NewClassPageHandler(classes, years, newClassTestTemplate(t))
	if err != nil {
		t.Fatal(err)
	}
	req := requestWithActor(http.MethodGet, "/classes?academic_year_id="+yearID.String(), service.Actor{UserID: uuid.New(), Username: "admin", Role: domain.RoleAdmin})
	w := httptest.NewRecorder()
	middleware.NewCSRFMiddleware(newTestCSRFCookie(t)).Protect(http.HandlerFunc(h.List)).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "VII A") || !strings.Contains(w.Body.String(), "2026/2027") {
		t.Fatalf("expected class and academic year in rendered page: %s", w.Body.String())
	}
	if classes.listInput.AcademicYearID != yearID {
		t.Fatalf("AcademicYearID = %s, want %s", classes.listInput.AcademicYearID, yearID)
	}
}

func TestClassPageCreateRedirectsAfterServiceSuccess(t *testing.T) {
	classes := &classServiceFake{}
	h, err := NewClassPageHandler(classes, &academicYearServiceFake{}, newClassTestTemplate(t))
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"name": {"VII A"}, "level": {"7"}, "csrf_token": {"test-token"}}
	req := httptest.NewRequest(http.MethodPost, "/classes", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Username: "admin", Role: domain.RoleAdmin}))
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	if w.Header().Get("Location") != "/classes" {
		t.Fatalf("Location = %q", w.Header().Get("Location"))
	}
	if classes.created.Name != "VII A" || classes.created.Level != 7 {
		t.Fatalf("service input = %+v", classes.created)
	}
}

func TestClassPageCreateRejectsInvalidLevel(t *testing.T) {
	h, err := NewClassPageHandler(&classServiceFake{}, &academicYearServiceFake{}, newClassTestTemplate(t))
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"name": {"VII A"}, "level": {"99"}, "csrf_token": {"test-token"}}
	req := httptest.NewRequest(http.MethodPost, "/classes", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}))
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "Nama kelas wajib diisi dan tingkat harus berupa angka 1 sampai 12.") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestClassPageUpdateRedirectsToDetail(t *testing.T) {
	id := uuid.New()
	classes := &classServiceFake{}
	h, err := NewClassPageHandler(classes, &academicYearServiceFake{}, newClassTestTemplate(t))
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"name": {"VII B"}, "level": {"7"}, "csrf_token": {"test-token"}}
	req := httptest.NewRequest(http.MethodPost, "/classes/"+id.String(), strings.NewReader(form.Encode()))
	req.SetPathValue("id", id.String())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}))
	w := httptest.NewRecorder()
	h.Update(w, req)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/classes/"+id.String() {
		t.Fatalf("status=%d location=%q", w.Code, w.Header().Get("Location"))
	}
	if classes.updated.ClassID != id || classes.updated.Name != "VII B" {
		t.Fatalf("update input = %+v", classes.updated)
	}
}

func newClassTestTemplate(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("classes").Parse(`{{define "classes"}}{{.Heading}} {{.CSRFToken}}{{range .AcademicYears}}{{.Name}}{{end}}{{range .Rows}}{{.Name}}{{end}}{{if .Error}}{{.Error}}{{end}}{{end}}`)
	if err != nil {
		t.Fatal(err)
	}
	return tmpl
}

func TestClassPageListRejectsInvalidPagination(t *testing.T) {
	yearID := uuid.New()
	for _, target := range []string{
		"/classes?academic_year_id=" + yearID.String() + "&page=0",
		"/classes?academic_year_id=" + yearID.String() + "&page=abc",
		"/classes?academic_year_id=" + yearID.String() + "&limit=0",
		"/classes?academic_year_id=" + yearID.String() + "&limit=101",
	} {
		t.Run(target, func(t *testing.T) {
			classes := &classServiceFake{}
			h, err := NewClassPageHandler(classes, &academicYearServiceFake{}, newClassTestTemplate(t))
			if err != nil {
				t.Fatal(err)
			}
			req := requestWithActor(http.MethodGet, target, service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin})
			recorder := httptest.NewRecorder()
			middleware.NewCSRFMiddleware(newTestCSRFCookie(t)).Protect(http.HandlerFunc(h.List)).ServeHTTP(recorder, req)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", recorder.Code, recorder.Body.String())
			}
			if classes.listInput.AcademicYearID != uuid.Nil {
				t.Fatalf("class service called for invalid pagination: %+v", classes.listInput)
			}
		})
	}
}

func TestClassPageListUsesLookaheadAtPaginationBoundary(t *testing.T) {
	yearID := uuid.New()
	classes := &classServiceFake{classes: []*domain.Class{
		{ID: uuid.New(), Name: "VII A", Level: 7},
		{ID: uuid.New(), Name: "VII B", Level: 7},
		{ID: uuid.New(), Name: "VII C", Level: 7},
	}}
	tmpl, err := template.New("classes").Parse(`{{define "classes"}}{{.HasPrevious}}|{{.HasNext}}|{{.Page}}|{{.Limit}}|{{len .Rows}}{{end}}`)
	if err != nil {
		t.Fatal(err)
	}
	h, err := NewClassPageHandler(classes, &academicYearServiceFake{}, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	req := requestWithActor(http.MethodGet, "/classes?academic_year_id="+yearID.String()+"&page=1&limit=2", service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin})
	recorder := httptest.NewRecorder()
	middleware.NewCSRFMiddleware(newTestCSRFCookie(t)).Protect(http.HandlerFunc(h.List)).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if got, want := recorder.Body.String(), "false|true|1|2|2"; got != want {
		t.Fatalf("view model = %q, want %q", got, want)
	}
	if classes.listInput.Options.Limit != 3 || classes.listInput.Options.Offset != 0 {
		t.Fatalf("service pagination = %+v, want limit 3 offset 0", classes.listInput.Options)
	}
}

func TestClassPageListMapsServiceFailure(t *testing.T) {
	yearID := uuid.New()
	classes := &classServiceFake{listErr: errors.New("internal repository detail")}
	h, err := NewClassPageHandler(classes, &academicYearServiceFake{}, newClassTestTemplate(t))
	if err != nil {
		t.Fatal(err)
	}
	req := requestWithActor(http.MethodGet, "/classes?academic_year_id="+yearID.String(), service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin})
	recorder := httptest.NewRecorder()
	middleware.NewCSRFMiddleware(newTestCSRFCookie(t)).Protect(http.HandlerFunc(h.List)).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "internal repository detail") {
		t.Fatalf("internal service detail leaked to response: %s", recorder.Body.String())
	}
}

func TestClassPageCreateRetainsSubmittedValuesOnServiceValidationError(t *testing.T) {
	classes := &classServiceFake{createErr: repository.ErrConflict}
	tmpl, err := template.New("classes").Parse(`{{define "classes"}}{{.Name}}|{{.Level}}|{{.CSRFToken}}|{{.Error}}{{end}}`)
	if err != nil {
		t.Fatal(err)
	}
	h, err := NewClassPageHandler(classes, &academicYearServiceFake{}, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"name": {"VII B"}, "level": {"8"}, "csrf_token": {"valid-token"}}
	req := httptest.NewRequest(http.MethodPost, "/classes", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}))
	recorder := httptest.NewRecorder()
	h.Create(recorder, req)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", recorder.Code, recorder.Body.String())
	}
	for _, want := range []string{"VII B", "8", "valid-token", "Nama kelas dengan tingkat tersebut sudah digunakan."} {
		if !strings.Contains(recorder.Body.String(), want) {
			t.Errorf("response %q does not contain %q", recorder.Body.String(), want)
		}
	}
}

func TestClassPageUpdateRetainsSubmittedValuesOnServiceValidationError(t *testing.T) {
	id := uuid.New()
	classes := &classServiceFake{updateErr: repository.ErrConflict}
	tmpl, err := template.New("classes").Parse(`{{define "classes"}}{{.ClassID}}|{{.Name}}|{{.Level}}|{{.CSRFToken}}|{{.Error}}{{end}}`)
	if err != nil {
		t.Fatal(err)
	}
	h, err := NewClassPageHandler(classes, &academicYearServiceFake{}, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"name": {"VII C"}, "level": {"9"}, "csrf_token": {"valid-token"}}
	req := httptest.NewRequest(http.MethodPost, "/classes/"+id.String(), strings.NewReader(form.Encode()))
	req.SetPathValue("id", id.String())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}))
	recorder := httptest.NewRecorder()
	h.Update(recorder, req)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", recorder.Code, recorder.Body.String())
	}
	for _, want := range []string{id.String(), "VII C", "9", "valid-token", "Nama kelas dengan tingkat tersebut sudah digunakan."} {
		if !strings.Contains(recorder.Body.String(), want) {
			t.Errorf("response %q does not contain %q", recorder.Body.String(), want)
		}
	}
}

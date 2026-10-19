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
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestNewTeacherAssignmentPageHandlerRejectsMissingDependencies(t *testing.T) {
	_, err := NewTeacherAssignmentPageHandler(nil, nil, nil, nil, nil)
	if err != service.ErrInvalidDependency {
		t.Fatalf("constructor error = %v, want %v", err, service.ErrInvalidDependency)
	}
}

func TestParseTeacherAssignmentFormRequiresFormContentType(t *testing.T) {
	req := httptest.NewRequest("POST", "/teacher-assignments", strings.NewReader("teacher_id=abc"))
	if err := parseTeacherAssignmentForm(httptest.NewRecorder(), req); err == nil {
		t.Fatal("expected missing content type to be rejected")
	}
}

func TestTeacherAssignmentListRendersYearScopedClassesAndCurrentAssignment(t *testing.T) {
	yearID, classID, teacherID := uuid.New(), uuid.New(), uuid.New()
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	year := &domain.AcademicYear{ID: yearID, Name: "2026/2027", StartDate: start, EndDate: time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC)}
	class := &domain.Class{ID: classID, Name: "VII A", Level: 7}
	assignment := &domain.TeacherClassAssignment{ID: uuid.New(), UserID: teacherID, ClassID: classID, AcademicYearID: yearID}
	assignments := &teacherAssignmentServiceFake{current: assignment}
	users := &teacherAssignmentUserServiceFake{users: []*domain.User{{ID: teacherID, Username: "wali_7a", PasswordHash: "must-not-render"}}}
	h, err := NewTeacherAssignmentPageHandler(assignments, users, &classServiceFake{classes: []*domain.Class{class}}, &academicYearServiceFake{years: []*domain.AcademicYear{year}}, newTeacherAssignmentTestTemplate(t))
	if err != nil {
		t.Fatal(err)
	}
	req := requestWithActor(http.MethodGet, "/teacher-assignments?academic_year_id="+yearID.String(), service.Actor{UserID: uuid.New(), Username: "admin", Role: domain.RoleAdmin})
	w := httptest.NewRecorder()
	middleware.NewCSRFMiddleware(newTestCSRFCookie(t)).Protect(http.HandlerFunc(h.List)).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "VII A") || !strings.Contains(w.Body.String(), "wali_7a") {
		t.Fatalf("expected class and assigned teacher in response: %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "must-not-render") {
		t.Fatalf("password hash leaked in response: %s", w.Body.String())
	}
	if !assignments.currentInput.Date.Equal(start) {
		t.Fatalf("assignment lookup date = %v, want academic-year start %v", assignments.currentInput.Date, start)
	}
}

func TestTeacherAssignmentCreateRejectsNonAdminBeforeCallingService(t *testing.T) {
	assignments := &teacherAssignmentServiceFake{}
	h, err := NewTeacherAssignmentPageHandler(
		assignments,
		&teacherAssignmentUserServiceFake{},
		&classServiceFake{},
		&academicYearServiceFake{},
		newTeacherAssignmentTestTemplate(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"teacher_id": {uuid.NewString()}, "class_id": {uuid.NewString()}, "academic_year_id": {uuid.NewString()}}
	req := httptest.NewRequest(http.MethodPost, "/teacher-assignments", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Username: "wali", Role: domain.RoleWaliKelas}))
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusForbidden, w.Body.String())
	}
	if assignments.assignCalled {
		t.Fatal("assignment service must not be called for non-admin")
	}
}

func TestTeacherAssignmentCreateRedirectsAfterSuccessfulAssignment(t *testing.T) {
	assignments := &teacherAssignmentServiceFake{}
	h, err := NewTeacherAssignmentPageHandler(
		assignments,
		&teacherAssignmentUserServiceFake{},
		&classServiceFake{},
		&academicYearServiceFake{},
		newTeacherAssignmentTestTemplate(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	teacherID, classID, yearID := uuid.New(), uuid.New(), uuid.New()
	form := url.Values{"teacher_id": {teacherID.String()}, "class_id": {classID.String()}, "academic_year_id": {yearID.String()}}
	req := httptest.NewRequest(http.MethodPost, "/teacher-assignments", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Username: "admin", Role: domain.RoleAdmin}))
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusSeeOther, w.Body.String())
	}
	if want := "/teacher-assignments?academic_year_id=" + yearID.String(); w.Header().Get("Location") != want {
		t.Fatalf("Location = %q, want %q", w.Header().Get("Location"), want)
	}
	if !assignments.assignCalled || assignments.assignInput.UserID != teacherID || assignments.assignInput.ClassID != classID || assignments.assignInput.AcademicYearID != yearID {
		t.Fatalf("assignment service input = %+v, called=%v", assignments.assignInput, assignments.assignCalled)
	}
}

func TestTeacherAssignmentFormErrorReturnsInternalErrorWhenOptionsCannotReload(t *testing.T) {
	years := &academicYearServiceFake{listErr: errors.New("database details must not leak")}
	h, err := NewTeacherAssignmentPageHandler(
		&teacherAssignmentServiceFake{},
		&teacherAssignmentUserServiceFake{},
		&classServiceFake{},
		years,
		newTeacherAssignmentTestTemplate(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"teacher_id": {"invalid"}, "class_id": {uuid.NewString()}, "academic_year_id": {uuid.NewString()}}
	req := httptest.NewRequest(http.MethodPost, "/teacher-assignments", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Username: "admin", Role: domain.RoleAdmin}))
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusInternalServerError, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "database details") {
		t.Fatalf("internal error leaked: %s", w.Body.String())
	}
}

func newTeacherAssignmentTestTemplate(t *testing.T) *template.Template {
	t.Helper()
	return template.Must(template.New("teacher-assignments").Parse(`{{define "teacher-assignments"}}{{.Heading}}{{if .Error}}{{.Error}}{{end}}{{range .AcademicYears}}{{.Name}}{{end}}{{range .Teachers}}{{.Name}}{{end}}{{range .Rows}}{{.ClassName}}{{end}}{{end}}`))
}

type teacherAssignmentServiceFake struct {
	assignCalled bool
	assignInput  service.AssignTeacherInput
	assignErr    error
	current      *domain.TeacherClassAssignment
	currentErr   error
	currentInput service.GetCurrentClassAssignmentInput
}

func (f *teacherAssignmentServiceFake) Assign(_ context.Context, in service.AssignTeacherInput) (service.AssignTeacherOutput, error) {
	f.assignCalled, f.assignInput = true, in
	return service.AssignTeacherOutput{}, f.assignErr
}
func (f *teacherAssignmentServiceFake) Get(context.Context, service.GetTeacherAssignmentInput) (service.GetTeacherAssignmentOutput, error) {
	return service.GetTeacherAssignmentOutput{}, nil
}
func (f *teacherAssignmentServiceFake) ListByTeacher(context.Context, service.ListTeacherAssignmentsInput) (service.ListTeacherAssignmentsOutput, error) {
	return service.ListTeacherAssignmentsOutput{}, nil
}
func (f *teacherAssignmentServiceFake) ListByClass(context.Context, service.ListClassAssignmentsInput) (service.ListClassAssignmentsOutput, error) {
	return service.ListClassAssignmentsOutput{}, nil
}
func (f *teacherAssignmentServiceFake) GetCurrentByClass(_ context.Context, in service.GetCurrentClassAssignmentInput) (service.GetCurrentClassAssignmentOutput, error) {
	f.currentInput = in
	return service.GetCurrentClassAssignmentOutput{Assignment: f.current}, f.currentErr
}

type teacherAssignmentUserServiceFake struct {
	users      []*domain.User
	listErr    error
	listCalled bool
}

func (f *teacherAssignmentUserServiceFake) Get(context.Context, service.GetUserInput) (service.GetUserOutput, error) {
	return service.GetUserOutput{}, nil
}
func (f *teacherAssignmentUserServiceFake) GetByUsername(context.Context, service.GetUserByUsernameInput) (service.GetUserByUsernameOutput, error) {
	return service.GetUserByUsernameOutput{}, nil
}
func (f *teacherAssignmentUserServiceFake) ListByRole(context.Context, service.ListUsersByRoleInput) (service.ListUsersByRoleOutput, error) {
	f.listCalled = true
	return service.ListUsersByRoleOutput{Users: f.users}, f.listErr
}
func (f *teacherAssignmentUserServiceFake) Create(context.Context, service.CreateUserInput) (service.CreateUserOutput, error) {
	return service.CreateUserOutput{}, nil
}
func (f *teacherAssignmentUserServiceFake) CreateStudentAccount(context.Context, service.CreateStudentAccountInput) (service.CreateStudentAccountOutput, error) {
	return service.CreateStudentAccountOutput{}, nil
}
func (f *teacherAssignmentUserServiceFake) Update(context.Context, service.UpdateUserInput) (service.UpdateUserOutput, error) {
	return service.UpdateUserOutput{}, nil
}

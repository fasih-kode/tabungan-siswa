package http

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type userPageUsersFake struct{ service.UserService }

func (userPageUsersFake) ListByRole(context.Context, service.ListUsersByRoleInput) (service.ListUsersByRoleOutput, error) {
	return service.ListUsersByRoleOutput{}, nil
}

type userPageStudentsFake struct {
	service.StudentService
	students   []*domain.Student
	placements map[uuid.UUID]service.StudentPlacement
}

func (f userPageStudentsFake) List(context.Context, service.ListStudentsInput) (service.ListStudentsOutput, error) {
	return service.ListStudentsOutput{Students: f.students, Placements: f.placements}, nil
}

func TestUsersPageLoadsUnlinkedStudentsWhileWaliKelasTabIsSelected(t *testing.T) {
	nis := "0012"
	linkedUserID := uuid.New()
	yearID := uuid.New()
	classID := uuid.New()
	handler := &UserPageHandler{
		users:         userPageUsersFake{},
		academicYears: &academicYearServiceFake{years: []*domain.AcademicYear{{ID: yearID, Name: "2026/2027", Status: domain.AcademicYearOpen}}},
		classes:       &classServiceFake{classes: []*domain.Class{{ID: classID, Name: "VII A", AcademicYearID: yearID}}},
		students: userPageStudentsFake{students: []*domain.Student{
			{ID: uuid.New(), NIS: &nis, Name: "Ani", Status: domain.StudentActive},
			{ID: uuid.New(), NIS: &nis, Name: "Budi", Status: domain.StudentLeft},
			{ID: uuid.New(), UserID: &linkedUserID, NIS: &nis, Name: "Cici", Status: domain.StudentActive},
		}},
		templates: template.Must(template.New("users").Parse(`{{define "users"}}{{.OpenAcademicYear}}|{{range .Classes}}<option value="{{.ID}}">{{.Label}}</option>{{end}}{{end}}`)),
	}
	request := requestWithActor(http.MethodGet, "/users?role=WALI_KELAS", service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin})
	response := httptest.NewRecorder()
	middleware.NewCSRFMiddleware(newTestCSRFCookie(t)).Protect(http.HandlerFunc(handler.List)).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, "2026/2027") || !strings.Contains(body, "VII A") {
		t.Fatalf("open-year class filter is missing from the account form: %s", body)
	}
}

func TestUserStudentOptionsAreFilteredByOpenYearClassAndAccountEligibility(t *testing.T) {
	yearID, classID := uuid.New(), uuid.New()
	linkedUserID := uuid.New()
	eligible := &domain.Student{ID: uuid.New(), NIS: stringPointer("0012"), Name: "Ani", Status: domain.StudentActive}
	linked := &domain.Student{ID: uuid.New(), UserID: &linkedUserID, NIS: stringPointer("0013"), Name: "Budi", Status: domain.StudentActive}
	inactive := &domain.Student{ID: uuid.New(), NIS: stringPointer("0014"), Name: "Cici", Status: domain.StudentLeft}
	handler := &UserPageHandler{
		academicYears: &academicYearServiceFake{years: []*domain.AcademicYear{{ID: yearID, Name: "2026/2027", Status: domain.AcademicYearOpen}}},
		classes:       &classServiceFake{classes: []*domain.Class{{ID: classID, Name: "VII A", AcademicYearID: yearID}}},
		students: userPageStudentsFake{
			students: []*domain.Student{eligible, linked, inactive},
			placements: map[uuid.UUID]service.StudentPlacement{
				eligible.ID: {AcademicYearID: yearID, ClassID: classID},
				linked.ID:   {AcademicYearID: yearID, ClassID: classID},
				inactive.ID: {AcademicYearID: yearID, ClassID: classID},
			},
		},
	}
	request := requestWithActor(http.MethodGet, "/users/student-options?class_id="+classID.String(), service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin})
	response := httptest.NewRecorder()
	handler.StudentOptions(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	var options []UnlinkedStudentOption
	if err := json.NewDecoder(response.Body).Decode(&options); err != nil {
		t.Fatal(err)
	}
	if len(options) != 1 || options[0].ID != eligible.ID.String() || options[0].Label != "Ani — 0012" {
		t.Fatalf("student options = %+v, want only eligible student", options)
	}
}

func stringPointer(value string) *string { return &value }

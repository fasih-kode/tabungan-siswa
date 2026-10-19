package http

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

type placementStudentServiceFake struct {
	service.StudentService
	created     service.CreateStudentInput
	createCalls int
}

func (f *placementStudentServiceFake) Create(_ context.Context, input service.CreateStudentInput) (service.CreateStudentOutput, error) {
	f.createCalls++
	f.created = input
	return service.CreateStudentOutput{Student: &domain.Student{ID: uuid.New()}}, nil
}
func newPlacementPageHandler(t *testing.T, students service.StudentService, classes service.ClassService) *StudentPageHandler {
	t.Helper()
	tmpl := template.Must(template.New("students").Parse(`{{define "students"}}{{.Error}}{{end}}`))
	h, err := NewStudentPageHandler(students, &academicYearServiceFake{}, classes, &teacherAssignmentUserServiceFake{}, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	return h
}
func TestStudentManualCreationPassesYearAndClassWithoutLogin(t *testing.T) {
	students := &placementStudentServiceFake{}
	h := newPlacementPageHandler(t, students, &classServiceFake{})
	year, class := uuid.New(), uuid.New()
	form := url.Values{"name": {"Siswa Baru"}, "nis": {"0012"}, "nisn": {"000034"}, "academic_year_id": {year.String()}, "class_id": {class.String()}, "user_id": {uuid.NewString()}}
	req := requestWithActor(http.MethodPost, "/students", service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin})
	req.Body = http.NoBody
	req = httptest.NewRequest(http.MethodPost, "/students", strings.NewReader(form.Encode())).WithContext(req.Context())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	h.Create(response, req)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if students.createCalls != 1 || students.created.AcademicYearID != year || students.created.ClassID != class || students.created.UserID != nil || *students.created.NIS != "0012" {
		t.Fatalf("create input=%+v", students.created)
	}
}
func TestStudentClassOptionsAreScopedAndAdminOnly(t *testing.T) {
	year, otherYear := uuid.New(), uuid.New()
	date := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	classes := &classServiceFake{classes: []*domain.Class{
		{ID: uuid.New(), Name: "VII A", AcademicYearID: year, EffectiveDate: date},
		{ID: uuid.New(), Name: "VIII A", AcademicYearID: otherYear, EffectiveDate: date},
		{ID: uuid.New(), Name: "Belum dikonfigurasi"},
	}}
	h := newPlacementPageHandler(t, &placementStudentServiceFake{}, classes)
	req := requestWithActor(http.MethodGet, "/students/class-options?academic_year_id="+year.String(), service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin})
	response := httptest.NewRecorder()
	h.ClassOptions(response, req)
	var options []StudentClassOption
	if err := json.Unmarshal(response.Body.Bytes(), &options); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || len(options) != 1 || options[0].Name != "VII A" || options[0].EffectiveDate != "2026-07-15" {
		t.Fatalf("options=%+v status=%d", options, response.Code)
	}
	for _, role := range []domain.UserRole{domain.RoleWaliKelas, domain.RoleSiswa} {
		req = requestWithActor(http.MethodGet, "/students/class-options?academic_year_id="+year.String(), service.Actor{UserID: uuid.New(), Role: role})
		response = httptest.NewRecorder()
		h.ClassOptions(response, req)
		if response.Code != http.StatusForbidden {
			t.Fatalf("role %s status=%d", role, response.Code)
		}
	}
}
func TestStudentWorkbookTemplateAndParserPreserveIdentitiesAndRows(t *testing.T) {
	h := newPlacementPageHandler(t, &placementStudentServiceFake{}, &classServiceFake{})
	req := requestWithActor(http.MethodGet, "/students/import/template", service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin})
	response := httptest.NewRecorder()
	h.ImportTemplate(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
	book, err := excelize.OpenReader(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	for row, values := range map[int][]any{2: {"Ani", "0012", "000034", "2026/2027", "VII A"}, 4: {"Budi", "0013", "000035", "2027/2028", "VIII A"}} {
		for column, value := range values {
			cell, _ := excelize.CoordinatesToCellName(column+1, row)
			if err := book.SetCellValue("Sheet1", cell, value); err != nil {
				t.Fatal(err)
			}
		}
	}
	buffer, err := book.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	rows, issues := parseStudentWorkbook(buffer)
	if len(issues) > 0 || len(rows) != 2 || rows[0].NIS != "0012" || rows[0].NISN != "000034" || rows[0].AcademicYear != "2026/2027" || rows[1].Class != "VIII A" || rows[1].SpreadsheetRow != 4 {
		t.Fatalf("rows=%+v issues=%+v", rows, issues)
	}
	if err := book.SetCellValue("Sheet1", "E1", ""); err != nil {
		t.Fatal(err)
	}
	buffer, err = book.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	_, issues = parseStudentWorkbook(buffer)
	if len(issues) != 1 || issues[0].SpreadsheetRow != 1 || !strings.Contains(issues[0].Message, "class") {
		t.Fatalf("missing-header issues=%+v", issues)
	}
}

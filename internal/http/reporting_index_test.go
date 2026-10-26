package http

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type reportingIndexServiceFake struct{ service.ReportingService }

func newReportingIndexHandlerForTest(t *testing.T) *ReportingPageHandler {
	t.Helper()
	tmpl := template.Must(template.New("reports").Parse(`{{define "reports"}}{{template "reports-content" .}}{{end}}{{define "reports-content"}}{{.Heading}}{{range .Links}}|{{.Icon}}:{{.Title}}={{.URL}}{{end}}{{end}}`))
	h, err := NewReportingPageHandler(reportingIndexServiceFake{}, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestReportingIndexShowsAllReportDestinations(t *testing.T) {
	h := newReportingIndexHandlerForTest(t)
	req := requestWithActor(http.MethodGet, "/reports", service.Actor{UserID: uuid.New(), Username: "admin", Role: domain.RoleAdmin})
	rec := httptest.NewRecorder()
	h.Index(rec, req)
	body := rec.Body.String()
	for _, expected := range []string{"students:Laporan Siswa=/reports/students", "classes:Laporan Kelas=/reports/classes", "academic-year:Laporan Tahun Ajaran=/reports/academic-year", "settlement:Laporan Penyelesaian Tabungan=/reports/settlements"} {
		if rec.Code != http.StatusOK || !strings.Contains(body, expected) {
			t.Fatalf("index missing %q: status=%d body=%s", expected, rec.Code, body)
		}
	}
}

func TestReportingIndexRejectsStudentRole(t *testing.T) {
	h := newReportingIndexHandlerForTest(t)
	req := requestWithActor(http.MethodGet, "/reports", service.Actor{UserID: uuid.New(), Username: "student", Role: domain.RoleSiswa})
	rec := httptest.NewRecorder()
	h.Index(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("student reports index status=%d body=%s", rec.Code, rec.Body.String())
	}
}

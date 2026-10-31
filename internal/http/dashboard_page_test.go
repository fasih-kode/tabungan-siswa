package http

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestDashboardRendersAuthenticatedHeaderContext(t *testing.T) {
	tmpl := template.Must(template.New("base").Parse(
		`{{define "base"}}{{.Username}}|{{.Role}}|{{.CSRFToken}}|{{.Heading}}{{end}}`,
	))
	handler, err := NewDashboardPageHandler(tmpl)
	if err != nil {
		t.Fatal(err)
	}

	actor := service.Actor{
		UserID:   uuid.New(),
		Username: "admin-test",
		Role:     domain.RoleAdmin,
	}
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req = req.WithContext(service.WithActor(req.Context(), actor))
	recorder := httptest.NewRecorder()

	middleware.NewCSRFMiddleware(newTestCSRFCookie(t)).
		Protect(http.HandlerFunc(handler.Get)).
		ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, want := range []string{"admin-test", "ADMIN", "Ringkasan"} {
		if !strings.Contains(body, want) {
			t.Errorf("rendered body %q does not contain %q", body, want)
		}
	}
	if strings.Count(body, "|") != 3 {
		t.Errorf("rendered body %q does not include username, role, CSRF token and heading", body)
	}
}

func TestDashboardMetricsCarryIconsForEachRole(t *testing.T) {
	metrics := repository.DashboardMetrics{Students: 18, Classes: 3, Accounts: 4, SettledAccounts: 2}
	adminMetrics := dashboardMetrics(domain.RoleAdmin, metrics)
	wantAdminIcons := []string{"students", "classes", "balance", "deposit", "withdrawal", "completed"}
	if len(adminMetrics) != len(wantAdminIcons) {
		t.Fatalf("admin metric count = %d, want %d", len(adminMetrics), len(wantAdminIcons))
	}
	for i, want := range wantAdminIcons {
		if adminMetrics[i].Icon != want {
			t.Errorf("admin metric %q icon = %q, want %q", adminMetrics[i].Label, adminMetrics[i].Icon, want)
		}
	}

	studentMetrics := dashboardMetrics(domain.RoleSiswa, metrics)
	wantStudentIcons := []string{"balance", "deposit", "withdrawal"}
	if len(studentMetrics) != len(wantStudentIcons) {
		t.Fatalf("student metric count = %d, want %d", len(studentMetrics), len(wantStudentIcons))
	}
	for i, want := range wantStudentIcons {
		if studentMetrics[i].Icon != want {
			t.Errorf("student metric %q icon = %q, want %q", studentMetrics[i].Label, studentMetrics[i].Icon, want)
		}
	}
}

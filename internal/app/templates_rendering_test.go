package app

import (
	"html/template"
	"strings"
	"testing"

	httpapp "github.com/fasih/tabungan-siswa/internal/http"
)

func TestAcademicYearTemplateRendersAcademicContentForAdminAndWali(t *testing.T) {
	templates, err := template.ParseFS(
		templateFiles,
		"templates/layouts/*.html",
		"templates/partials/*.html",
		"templates/auth/*.html",
		"templates/pages/*.html",
	)
	if err != nil {
		t.Fatalf("ParseFS() error = %v", err)
	}

	for _, tc := range []struct {
		name string
		role string
		want string
	}{
		{name: "admin", role: "ADMIN", want: "Tambah Tahun Ajaran"},
		{name: "wali kelas", role: "WALI_KELAS", want: "Belum ada tahun ajaran."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			view := httpapp.AcademicYearPageViewModel{
				Title: "Tahun Ajaran", Heading: "Tahun Ajaran",
				Description: "Kelola tahun ajaran", Role: tc.role, Username: "test-user",
				Page: 1, Limit: 20,
			}
			var output strings.Builder
			if err := templates.ExecuteTemplate(&output, "academic-years", view); err != nil {
				t.Fatalf("ExecuteTemplate() error = %v", err)
			}
			body := output.String()
			if !strings.Contains(body, tc.want) {
				t.Fatalf("academic-year page does not contain %q", tc.want)
			}
			if strings.Contains(body, "Aktivitas Terbaru") || strings.Contains(body, "Belum ada aktivitas") {
				t.Fatalf("academic-year page rendered dashboard content")
			}
		})
	}
}

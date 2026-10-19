package app

import (
	"html/template"
	"strings"
	"testing"

	httpapp "github.com/fasih/tabungan-siswa/internal/http"
	"github.com/fasih/tabungan-siswa/internal/service"
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

func TestChangePasswordTemplateRendersForStudent(t *testing.T) {
	templates, err := template.ParseFS(templateFiles, "templates/layouts/*.html", "templates/partials/*.html", "templates/auth/*.html", "templates/pages/*.html")
	if err != nil {
		t.Fatal(err)
	}
	view := httpapp.UserPageViewModel{Title: "Ganti Password", Heading: "Ganti Password", Role: "SISWA", Username: "siswa-test", CSRFToken: "csrf-test", ActiveNav: "change-password"}
	var out strings.Builder
	if err := templates.ExecuteTemplate(&out, "change-password", view); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	for _, want := range []string{"Ganti Password", "siswa-test", `action="/account/password"`, `name="current_password"`, `name="new_password"`, `name="confirm_password"`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("change-password page missing %q", want)
		}
	}
}

func TestStudentDirectoryRendersTableAndPlacementModal(t *testing.T) {
	templates, err := template.ParseFS(templateFiles, "templates/layouts/*.html", "templates/partials/*.html", "templates/auth/*.html", "templates/pages/*.html")
	if err != nil {
		t.Fatal(err)
	}
	view := httpapp.StudentPageViewModel{
		Title: "Siswa", Heading: "Daftar Siswa", Role: "ADMIN", IsAdmin: true, Page: 1,
		Rows: []httpapp.StudentRow{{ID: "student", Name: "Ani", NIS: "001", NISN: "0001", AcademicYear: "2026/2027", Class: "VII A", Status: "ACTIVE"}},
	}
	var out strings.Builder
	if err := templates.ExecuteTemplate(&out, "students", view); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	for _, want := range []string{`data-modal-open="student-modal"`, `name="academic_year_id"`, `name="class_id"`, "2026/2027", "VII A", "academic_year, dan class"} {
		if !strings.Contains(body, want) {
			t.Errorf("student page missing %q", want)
		}
	}
	if strings.Index(body, "<table") > strings.Index(body, "Impor siswa dari Excel") {
		t.Fatal("directory should appear before import form")
	}
	if strings.Contains(body, `name="effective_date"`) || strings.Contains(body, `name="user_id"`) {
		t.Fatal("creation asks for per-student date or login link")
	}
	view.Error = "NIS sudah digunakan"
	out.Reset()
	if err := templates.ExecuteTemplate(&out, "students", view); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "data-modal-show") {
		t.Fatal("validation error does not reopen modal")
	}
}

func TestStudentDetailShowsTransferLinkOnlyToWaliKelas(t *testing.T) {
	templates, err := template.ParseFS(templateFiles, "templates/layouts/*.html", "templates/partials/*.html", "templates/auth/*.html", "templates/pages/*.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		role string
		want bool
	}{{"ADMIN", false}, {"WALI_KELAS", true}} {
		view := httpapp.StudentPageViewModel{Title: "Detail Siswa", Heading: "Detail Siswa", Role: tc.role, IsDetail: true, StudentID: "student", Name: "Ani", StudentStatus: "ACTIVE", TransferURL: "/students/student/class-transfer-requests?academic_year_id=year"}
		var out strings.Builder
		if err := templates.ExecuteTemplate(&out, "students", view); err != nil {
			t.Fatal(err)
		}
		got := strings.Contains(out.String(), "Ajukan transfer kelas")
		if got != tc.want {
			t.Errorf("role %s transfer button visible=%v, want %v", tc.role, got, tc.want)
		}
	}
}

func TestStudentPromotionTemplateRendersDecisions(t *testing.T) {
	templates, err := template.ParseFS(templateFiles, "templates/layouts/*.html", "templates/partials/*.html", "templates/auth/*.html", "templates/pages/*.html")
	if err != nil {
		t.Fatal(err)
	}
	view := httpapp.StudentPromotionPageViewModel{
		Title: "Kenaikan Kelas", Heading: "Kenaikan Kelas dan Kelulusan", Role: "ADMIN", FromYearID: "from", ToYearID: "to", CSRFToken: "token",
		Candidates: []httpapp.PromotionCandidateRow{{StudentPromotionCandidate: service.StudentPromotionCandidate{StudentID: "student", StudentName: "Ani", FromClassID: "class7", FromClassName: "VII A", FromLevel: 7, DefaultTargetID: "__graduate__", GraduationCandidate: true}, Targets: []httpapp.PromotionTargetOption{{ID: "class7b", Name: "VII B (tingkat 7)"}}}},
		Excluded:   []service.StudentPromotionCandidate{{StudentName: "Budi", Status: "LEFT"}},
	}
	var out strings.Builder
	if err := templates.ExecuteTemplate(&out, "student-promotions", view); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	for _, want := range []string{"name=\"decision\"", "__graduate__", "Proses Kenaikan Kelas", "Tidak ikut diproses", "Budi", "Kenaikan & Kelulusan"} {
		if !strings.Contains(body, want) {
			t.Errorf("promotion page missing %q", want)
		}
	}
}

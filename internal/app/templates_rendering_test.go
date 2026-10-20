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
			if tc.role == "ADMIN" && !strings.Contains(body, "Tahun baru dapat dibuat setelah seluruh tahun sebelumnya ditutup") {
				t.Fatal("academic-year page does not explain the year-creation prerequisite")
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

func TestSavingsTemplateRendersAccountCreationAndPassbook(t *testing.T) {
	templates, err := template.ParseFS(templateFiles, "templates/layouts/*.html", "templates/partials/*.html", "templates/auth/*.html", "templates/pages/*.html")
	if err != nil {
		t.Fatal(err)
	}
	list := httpapp.SavingsPageViewModel{Title: "Tabungan", Heading: "Tabungan Siswa", Description: "Siklus", Role: "WALI_KELAS", Username: "wali", CSRFToken: "csrf", AcademicYearID: "year", Years: []httpapp.SavingsYearOption{{ID: "year", Name: "2026/2027"}}, Rows: []httpapp.SavingsDirectoryRow{{StudentID: "student", Name: "Ani", StudentStatus: "ACTIVE", CanCreate: true}}}
	var out strings.Builder
	if err := templates.ExecuteTemplate(&out, "savings", list); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"2026/2027", "Buka tabungan", `action="/savings"`, `name="csrf_token"`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("savings list missing %q", want)
		}
	}
	detail := httpapp.SavingsPageViewModel{Title: "Tabungan", Heading: "Tabungan Siswa", Description: "Siklus", Role: "WALI_KELAS", Username: "wali", CSRFToken: "csrf", IsDetail: true, CanTransact: true, AccountID: "account", StudentName: "Ani", AccountStatus: "OPEN", YearName: "2026/2027", Balance: "Rp 15.000", TransactionURL: "/savings/account/transactions", TransactionDate: "2026-08-01", MinTransactionDate: "2026-07-01", MaxTransactionDate: "2027-06-30", Passbook: []httpapp.SavingsPassbookRow{{Date: "2026-08-01", Type: "Setor", Amount: "Rp 15.000", Balance: "Rp 15.000"}}}
	out.Reset()
	if err := templates.ExecuteTemplate(&out, "savings", detail); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Buku", "Saldo berjalan", "2026-08-01", "Rp 15.000", "action=\"/savings/account/transactions/deposits\"", "action=\"/savings/account/transactions/withdrawals\"", "name=\"csrf_token\""} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("passbook detail missing %q", want)
		}
	}
}

func TestTransactionTemplateRendersListAndDetail(t *testing.T) {
	templates, err := template.ParseFS(templateFiles, "templates/layouts/*.html", "templates/partials/*.html", "templates/auth/*.html", "templates/pages/*.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, view := range []httpapp.TransactionPageViewModel{
		{Title: "Transaksi", Heading: "Transaksi Tabungan", Role: "ADMIN", Username: "admin", AccountID: "account", StudentName: "Ani", YearName: "2026/2027", AccountStatus: "OPEN", AccountBalance: "Rp 42.000", Rows: []httpapp.TransactionPageRow{{ID: "transaction", Date: "2026-10-01", Type: "Setoran", Amount: "Rp 42.000", Balance: "Rp 42.000"}}},
		{Title: "Transaksi", Heading: "Transaksi Tabungan", Role: "SISWA", Username: "siswa", IsDetail: true, TransactionID: "transaction", AccountID: "account", StudentName: "Ani", YearName: "2026/2027", TransactionDate: "2026-10-01", TransactionType: "Setoran", Amount: "Rp 42.000", RecordedAt: "2026-10-01T10:00:00Z"},
	} {
		var out strings.Builder
		if err := templates.ExecuteTemplate(&out, "transactions", view); err != nil {
			t.Fatal(err)
		}
		body := out.String()
		for _, want := range []string{"Ani", "2026/2027", "Detail Transaksi", "Riwayat Transaksi"} {
			// The list and detail have separate content by design.
			if want == "Detail Transaksi" && !view.IsDetail || want == "Riwayat Transaksi" && view.IsDetail {
				continue
			}
			if !strings.Contains(body, want) {
				t.Errorf("transaction template missing %q for detail=%v", want, view.IsDetail)
			}
		}
	}
}

func TestSavingsTemplateEmptyAndSettledStates(t *testing.T) {
	templates, err := template.ParseFS(templateFiles, "templates/layouts/*.html", "templates/partials/*.html", "templates/auth/*.html", "templates/pages/*.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"OPEN", "SETTLED"} {
		t.Run(status, func(t *testing.T) {
			vm := httpapp.SavingsPageViewModel{Title: "Tabungan", Role: "ADMIN", IsDetail: true, StudentName: "Ani", AccountStatus: status, Balance: "Rp 0"}
			var out strings.Builder
			if err := templates.ExecuteTemplate(&out, "savings", vm); err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"Siklus " + status, "Rp 0", "Belum ada transaksi pada siklus tabungan ini."} {
				if !strings.Contains(out.String(), want) {
					t.Errorf("missing %q", want)
				}
			}
			if strings.Contains(out.String(), "Buka tabungan") {
				t.Error("detail must not offer account creation")
			}
		})
	}
	t.Run("settled directory", func(t *testing.T) {
		vm := httpapp.SavingsPageViewModel{Title: "Tabungan", Role: "ADMIN", AcademicYearID: "year", Rows: []httpapp.SavingsDirectoryRow{{Name: "Ani", AccountID: "settled-account", AccountStatus: "SETTLED", Balance: "Rp 0"}}}
		var out strings.Builder
		if err := templates.ExecuteTemplate(&out, "savings", vm); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"SETTLED", `href="/savings/settled-account"`} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("missing %q", want)
			}
		}
		if strings.Contains(out.String(), "Buka tabungan") {
			t.Error("settled cycle must remain readable without recreation")
		}
	})
	t.Run("empty directory", func(t *testing.T) {
		vm := httpapp.SavingsPageViewModel{Title: "Tabungan", Role: "ADMIN", AcademicYearID: "year"}
		var out strings.Builder
		if err := templates.ExecuteTemplate(&out, "savings", vm); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "Tidak ada siswa dalam cakupan tahun ajaran ini.") {
			t.Error("empty directory message missing")
		}
	})
}

func TestSettlementTemplateRendersActorNamesAndHistory(t *testing.T) {
	templates, err := template.ParseFS(templateFiles, "templates/layouts/*.html", "templates/partials/*.html", "templates/auth/*.html", "templates/pages/*.html")
	if err != nil {
		t.Fatal(err)
	}
	view := httpapp.SettlementPageVM{
		Title: "Detail Settlement", Heading: "Detail Settlement", Role: "ADMIN", Username: "admin-test",
		IsDetail: true, StudentName: "Siswa Uji", SettlementType: "REGULAR_YEAR_END", Amount: "Rp 12.500",
		ExecutedBy: "admin-pelaksana", ExecutedAt: "2026-10-20T10:00:00Z",
		History:   []httpapp.SettlementHistoryVM{{Type: "REGULAR_YEAR_END", Amount: "Rp 12.500", Status: "COMPLETED", ExecutedBy: "admin-pelaksana", ExecutedAt: "2026-10-20T10:00:00Z"}},
		Movements: []httpapp.SettlementMovementVM{{Kind: "HANDOVER", Amount: "Rp 12.500", Actor: "wali-penerima", At: "2026-10-20T10:01:00Z"}},
	}
	var out strings.Builder
	if err := templates.ExecuteTemplate(&out, "settlement-page", view); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	for _, want := range []string{"Siswa Uji", "admin-pelaksana", "Riwayat settlement akun", "wali-penerima"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("settlement template does not contain %q", want)
		}
	}
}

func TestSettlementPreviewShowsStaleBalanceWarning(t *testing.T) {
	templates, err := template.ParseFS(templateFiles, "templates/layouts/*.html", "templates/partials/*.html", "templates/auth/*.html", "templates/pages/*.html")
	if err != nil {
		t.Fatal(err)
	}
	view := httpapp.SettlementPageVM{
		Title: "Pratinjau Settlement", Heading: "Konfirmasi Settlement", Role: "WALI_KELAS", Username: "wali-test",
		IsPreview: true, CanSettle: true, StudentName: "Siswa Uji", Amount: "Rp 23.000", ExpectedBalance: "23000",
		PreviewedAt: "2026-10-20T10:05:00Z", Error: "Saldo berubah sejak pratinjau.",
	}
	var out strings.Builder
	if err := templates.ExecuteTemplate(&out, "settlement-page", view); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	for _, want := range []string{"Saldo berubah sejak pratinjau.", "Rp 23.000", `name="expected_balance" value="23000"`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("settlement preview does not contain %q", want)
		}
	}
}

func TestSettlementOverviewTemplateEmptyReadyAndBlockedStates(t *testing.T) {
	templates, err := template.ParseFS(templateFiles, "templates/layouts/*.html", "templates/partials/*.html", "templates/auth/*.html", "templates/pages/*.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		view httpapp.SettlementPageVM
		want string
	}{
		{name: "empty", view: httpapp.SettlementPageVM{Title: "Pengawasan Tabungan", Heading: "Kesiapan", Role: "ADMIN", AcademicYearID: "year", TransferURL: "/class-transfer-requests/pending?academic_year_id=year", ReadinessLabel: "Kesiapan tahun ajaran"}, want: "Belum ada rekening dalam cakupan."},
		{name: "admin ready", view: httpapp.SettlementPageVM{Title: "Pengawasan Tabungan", Heading: "Kesiapan", Role: "ADMIN", AcademicYearID: "year", TransferURL: "/class-transfer-requests/pending?academic_year_id=year", ReadinessLabel: "Kesiapan tahun ajaran", Ready: true}, want: "Siap ditutup"},
		{name: "wali blocked", view: httpapp.SettlementPageVM{Title: "Pengawasan Tabungan", Heading: "Kesiapan", Role: "WALI_KELAS", AcademicYearID: "year", TransferURL: "/class-transfer-requests/pending?academic_year_id=year", ReadinessLabel: "Kesiapan cakupan Anda", PendingTransfers: 1}, want: "Belum siap"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := templates.ExecuteTemplate(&out, "settlement-page", tc.view); err != nil {
				t.Fatalf("ExecuteTemplate() error = %v", err)
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Fatalf("template output does not contain %q", tc.want)
			}
		})
	}
}

func TestSettlementDetailTemplateIsReadOnlyForStudent(t *testing.T) {
	templates, err := template.ParseFS(templateFiles, "templates/layouts/*.html", "templates/partials/*.html", "templates/auth/*.html", "templates/pages/*.html")
	if err != nil {
		t.Fatal(err)
	}
	view := httpapp.SettlementPageVM{
		Title: "Detail Settlement", Heading: "Detail Settlement", Role: "SISWA", Username: "siswa-test",
		IsDetail: true, StudentName: "Siswa Uji", SettlementID: "settlement-id", Amount: "Rp 12.500",
		History: []httpapp.SettlementHistoryVM{{Type: "REGULAR_YEAR_END", Amount: "Rp 12.500", Status: "COMPLETED", ExecutedBy: "wali-test", ExecutedAt: "2026-10-20T10:00:00Z"}},
	}
	var out strings.Builder
	if err := templates.ExecuteTemplate(&out, "settlement-page", view); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	body := out.String()
	for _, want := range []string{"Siswa Uji", "Riwayat settlement akun", "wali-test"} {
		if !strings.Contains(body, want) {
			t.Errorf("student settlement detail does not contain %q", want)
		}
	}
	for _, forbidden := range []string{"distribution-confirmation", "reopen-requests", "Mulai jendela koreksi", "Ajukan permintaan"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("student settlement detail contains write control %q", forbidden)
		}
	}
}

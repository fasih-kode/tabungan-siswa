package http

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/service"
)

type ReportTableRow struct{ Cells []string }
type ReportLink struct{ Title, Description, URL, Icon string }
type ReportingPageViewModel struct {
	Title, Heading, Description, Username, Role, CSRFToken, ActiveNav, AcademicYear string
	Headers                                                                         []string
	Rows                                                                            []ReportTableRow
	HasActiveYear                                                                   bool
	IsIndex                                                                         bool
	Links                                                                           []ReportLink
}

func (h *ReportingPageHandler) Index(w http.ResponseWriter, r *http.Request) {
	actor, ok := reportActor(w, r)
	if !ok {
		return
	}
	if err := service.RequireRole(actor, domain.RoleAdmin, domain.RoleWaliKelas); err != nil {
		writeReportingHTTPError(w, err)
		return
	}
	vm := ReportingPageViewModel{Title: "Laporan", Heading: "Laporan", Description: "Pilih ringkasan yang ingin dilihat. Laporan menggunakan tahun ajaran aktif.", Username: actor.Username, Role: string(actor.Role), ActiveNav: "reports", IsIndex: true, Links: []ReportLink{
		{Title: "Laporan Siswa", Description: "Saldo, status rekening, dan total setor/tarik per siswa.", URL: "/reports/students", Icon: "students"},
		{Title: "Laporan Kelas", Description: "Jumlah siswa dan ringkasan tabungan per kelas.", URL: "/reports/classes", Icon: "classes"},
		{Title: "Laporan Tahun Ajaran", Description: "Ringkasan keseluruhan untuk tahun ajaran aktif.", URL: "/reports/academic-year", Icon: "academic-year"},
		{Title: "Laporan Penyelesaian Tabungan", Description: "Status penyelesaian tabungan dan penyerahan uang per siswa.", URL: "/reports/settlements", Icon: "settlement"},
	}}
	h.render(w, r, vm)
}

type ReportingPageHandler struct {
	reporting service.ReportingService
	templates *template.Template
}

func NewReportingPageHandler(reporting service.ReportingService, templates *template.Template) (*ReportingPageHandler, error) {
	if reporting == nil || templates == nil {
		return nil, service.ErrInvalidDependency
	}
	return &ReportingPageHandler{reporting: reporting, templates: templates}, nil
}

func (h *ReportingPageHandler) Students(w http.ResponseWriter, r *http.Request) {
	actor, ok := reportActor(w, r)
	if !ok {
		return
	}
	out, err := h.reporting.StudentReport(r.Context(), actor)
	if err != nil {
		writeReportingHTTPError(w, err)
		return
	}
	vm := h.base(actor, "Laporan Siswa", out.AcademicYear)
	vm.Headers = []string{"NIS", "Nama", "Kelas", "Status rekening", "Saldo", "Total setor", "Total tarik"}
	for _, x := range out.Rows {
		vm.Rows = append(vm.Rows, ReportTableRow{[]string{x.NIS, x.StudentName, x.ClassName, accountStatusLabel(x.AccountStatus), formatRupiah(x.Balance), formatRupiah(x.Deposits), formatRupiah(x.Withdrawals)}})
	}
	h.render(w, r, vm)
}

func (h *ReportingPageHandler) Classes(w http.ResponseWriter, r *http.Request) {
	actor, ok := reportActor(w, r)
	if !ok {
		return
	}
	out, err := h.reporting.ClassReport(r.Context(), actor)
	if err != nil {
		writeReportingHTTPError(w, err)
		return
	}
	vm := h.base(actor, "Laporan Kelas", out.AcademicYear)
	vm.Headers = []string{"Kelas", "Siswa", "Rekening", "Tabungan selesai", "Saldo", "Total setor", "Total tarik"}
	for _, x := range out.Rows {
		vm.Rows = append(vm.Rows, ReportTableRow{[]string{x.ClassName, reportInt(x.Students), reportInt(x.Accounts), reportInt(x.Settled), formatRupiah(x.Balance), formatRupiah(x.Deposits), formatRupiah(x.Withdrawals)}})
	}
	h.render(w, r, vm)
}

func (h *ReportingPageHandler) AcademicYear(w http.ResponseWriter, r *http.Request) {
	actor, ok := reportActor(w, r)
	if !ok {
		return
	}
	out, err := h.reporting.AcademicYearReport(r.Context(), actor)
	if err != nil {
		writeReportingHTTPError(w, err)
		return
	}
	vm := h.base(actor, "Laporan Tahun Ajaran", out.AcademicYear)
	vm.Headers = []string{"Siswa", "Kelas", "Rekening", "Tabungan selesai", "Saldo", "Total setor", "Total tarik"}
	if out.AcademicYear != nil {
		m := out.Metrics
		vm.Rows = []ReportTableRow{{[]string{reportInt(m.Students), reportInt(m.Classes), reportInt(m.Accounts), reportInt(m.SettledAccounts), formatRupiah(m.Balance), formatRupiah(m.Deposits), formatRupiah(m.Withdrawals)}}}
	}
	h.render(w, r, vm)
}

func (h *ReportingPageHandler) Settlements(w http.ResponseWriter, r *http.Request) {
	actor, ok := reportActor(w, r)
	if !ok {
		return
	}
	out, err := h.reporting.SettlementReport(r.Context(), actor)
	if err != nil {
		writeReportingHTTPError(w, err)
		return
	}
	vm := h.base(actor, "Laporan Penyelesaian Tabungan", out.AcademicYear)
	vm.Headers = []string{"NIS", "Nama", "Kelas", "Status penyelesaian tabungan", "Saldo", "Nilai penyelesaian", "Penyerahan tercatat"}
	for _, x := range out.Rows {
		vm.Rows = append(vm.Rows, ReportTableRow{[]string{x.NIS, x.StudentName, x.ClassName, settlementStatusLabel(x.SettlementState), formatRupiah(x.Balance), formatRupiah(x.SettlementAmount), formatRupiah(x.HandedOver)}})
	}
	h.render(w, r, vm)
}

func reportActor(w http.ResponseWriter, r *http.Request) (service.Actor, bool) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return service.Actor{}, false
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return service.Actor{}, false
	}
	return actor, true
}

func (h *ReportingPageHandler) base(actor service.Actor, title string, year *domain.AcademicYear) ReportingPageViewModel {
	vm := ReportingPageViewModel{Title: title, Heading: title, Description: "Laporan hanya menampilkan data tahun ajaran aktif.", Username: actor.Username, Role: string(actor.Role), ActiveNav: "reports"}
	if year != nil {
		vm.AcademicYear, vm.HasActiveYear = year.Name, true
	} else {
		vm.Description = "Tidak ada tahun ajaran aktif. Laporan belum tersedia."
	}
	return vm
}

func (h *ReportingPageHandler) render(w http.ResponseWriter, r *http.Request, vm ReportingPageViewModel) {
	if token, ok := middleware.CSRFToken(r); ok {
		vm.CSRFToken = token
	}
	if err := h.templates.ExecuteTemplate(w, "reports", vm); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func reportInt(value int) string { return fmt.Sprint(value) }
func accountStatusLabel(status domain.SavingsAccountStatus) string {
	switch status {
	case domain.SavingsAccountOpen:
		return "Terbuka"
	case domain.SavingsAccountSettled:
		return "Selesai"
	default:
		return "Belum ada rekening"
	}
}

func settlementStatusLabel(status string) string {
	switch status {
	case "OUTSTANDING":
		return "Belum diselesaikan"
	case "AWAITING_HANDOVER":
		return "Menunggu penyerahan"
	case "COMPLETED":
		return "Selesai"
	default:
		return "Perlu pemeriksaan"
	}
}

package http

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/httperror"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
)

type DashboardPageViewModel struct {
	Title          string
	Heading        string
	Description    string
	Username       string
	Role           string
	CSRFToken      string
	ActiveNav      string
	AcademicYear   string
	HasActiveYear  bool
	Metrics        []DashboardMetricView
	Recent         []DashboardTransactionView
	CanViewReports bool
}

func writeReportingHTTPError(w http.ResponseWriter, err error) { httperror.Write(w, err) }

type DashboardMetricView struct{ Label, Value, Icon string }
type DashboardTransactionView struct{ Date, Type, Amount, Status string }

type DashboardPageHandler struct {
	templates *template.Template
	reporting service.ReportingService
}

func NewDashboardPageHandler(templates *template.Template) (*DashboardPageHandler, error) {
	if templates == nil {
		return nil, service.ErrInvalidDependency
	}

	return &DashboardPageHandler{
		templates: templates,
	}, nil
}

func NewDashboardPageHandlerWithService(templates *template.Template, reporting service.ReportingService) (*DashboardPageHandler, error) {
	if templates == nil || reporting == nil {
		return nil, service.ErrInvalidDependency
	}
	return &DashboardPageHandler{templates: templates, reporting: reporting}, nil
}

func (h *DashboardPageHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(
			w,
			localizedStatusMessage(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed,
		)
		return
	}

	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	csrfToken, ok := middleware.CSRFToken(r)
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	viewModel := DashboardPageViewModel{
		Title:       "Ringkasan",
		Heading:     "Ringkasan",
		Description: "Ringkasan aplikasi Tabungan Siswa.",
		Username:    actor.Username,
		Role:        string(actor.Role),
		CSRFToken:   csrfToken,
		ActiveNav:   "dashboard",
	}
	if h.reporting != nil {
		out, err := h.reporting.Dashboard(r.Context(), actor)
		if err != nil {
			writeReportingHTTPError(w, err)
			return
		}
		if out.AcademicYear != nil {
			viewModel.AcademicYear, viewModel.HasActiveYear = out.AcademicYear.Name, true
			viewModel.Metrics = dashboardMetrics(actor.Role, out.Metrics)
			for _, transaction := range out.Recent {
				viewModel.Recent = append(viewModel.Recent, DashboardTransactionView{Date: transaction.TransactionDate.Format("2006-01-02"), Type: transactionTypeLabel(transaction.Type), Amount: formatRupiah(transaction.Amount), Status: interfaceStatusLabel(string(transaction.Status))})
			}
		} else {
			viewModel.Description = "Belum ada tahun ajaran aktif untuk ditampilkan."
		}
		viewModel.CanViewReports = actor.Role == domain.RoleAdmin || actor.Role == domain.RoleWaliKelas
	}

	templateName := "dashboard"
	if h.templates.Lookup(templateName) == nil {
		templateName = "base"
	}
	if err := h.templates.ExecuteTemplate(w, templateName, viewModel); err != nil {
		http.Error(
			w,
			localizedStatusMessage(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
	}
}

func dashboardMetrics(role domain.UserRole, metrics repository.DashboardMetrics) []DashboardMetricView {
	if role == domain.RoleSiswa {
		return []DashboardMetricView{
			{"Saldo", formatRupiah(metrics.Balance), "balance"},
			{"Total setor", formatRupiah(metrics.Deposits), "deposit"},
			{"Total tarik", formatRupiah(metrics.Withdrawals), "withdrawal"},
		}
	}
	return []DashboardMetricView{
		{"Siswa", strconv.Itoa(metrics.Students), "students"},
		{"Kelas", strconv.Itoa(metrics.Classes), "classes"},
		{"Saldo", formatRupiah(metrics.Balance), "balance"},
		{"Total setor", formatRupiah(metrics.Deposits), "deposit"},
		{"Total tarik", formatRupiah(metrics.Withdrawals), "withdrawal"},
		{"Rekening selesai", fmt.Sprintf("%d / %d", metrics.SettledAccounts, metrics.Accounts), "completed"},
	}
}

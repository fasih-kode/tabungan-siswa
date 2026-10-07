package http

import (
	"html/template"
	"net/http"

	"github.com/fasih/tabungan-siswa/internal/service"
)

type DashboardPageViewModel struct {
	Title       string
	Heading     string
	Description string
}

type DashboardPageHandler struct {
	templates *template.Template
}

func NewDashboardPageHandler(templates *template.Template) (*DashboardPageHandler, error) {
	if templates == nil {
		return nil, service.ErrInvalidDependency
	}

	return &DashboardPageHandler{
		templates: templates,
	}, nil
}

func (h *DashboardPageHandler) Get(w http.ResponseWriter, r *http.Request) {
	viewModel := DashboardPageViewModel{
		Title:       "Dashboard",
		Heading:     "Dashboard",
		Description: "Ringkasan aplikasi Tabungan Siswa.",
	}

	if err := h.templates.ExecuteTemplate(w, "base", viewModel); err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
	}
}

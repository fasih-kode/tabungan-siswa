package http

import (
	"html/template"
	"net/http"

	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/service"
)

type DashboardPageViewModel struct {
	Title       string
	Heading     string
	Description string
	Username    string
	Role        string
	CSRFToken   string
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
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(
			w,
			http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed,
		)
		return
	}

	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	csrfToken, ok := middleware.CSRFToken(r)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	viewModel := DashboardPageViewModel{
		Title:       "Dashboard",
		Heading:     "Dashboard",
		Description: "Ringkasan aplikasi Tabungan Siswa.",
		Username:    actor.Username,
		Role:        string(actor.Role),
		CSRFToken:   csrfToken,
	}

	if err := h.templates.ExecuteTemplate(w, "base", viewModel); err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
	}
}

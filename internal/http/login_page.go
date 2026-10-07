package http

import (
	"html/template"
	"net/http"

	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/service"
)

type LoginPageViewModel struct {
	Title     string
	CSRFToken string
}

type LoginPageHandler struct {
	templates *template.Template
}

func NewLoginPageHandler(templates *template.Template) (*LoginPageHandler, error) {
	if templates == nil {
		return nil, service.ErrInvalidDependency
	}

	return &LoginPageHandler{
		templates: templates,
	}, nil
}

func (h *LoginPageHandler) Get(w http.ResponseWriter, r *http.Request) {
	csrfToken, ok := middleware.CSRFToken(r)
	if !ok {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	viewModel := LoginPageViewModel{
		Title:     "Login",
		CSRFToken: csrfToken,
	}

	if err := h.templates.ExecuteTemplate(w, "auth", viewModel); err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
	}
}

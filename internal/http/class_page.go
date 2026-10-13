package http

import (
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/httperror"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

const maxClassFormBytes = 16 << 10

type ClassRowViewModel struct {
	ID    string
	Name  string
	Level int
}

type ClassAcademicYearOption struct {
	ID   string
	Name string
}

type ClassPageViewModel struct {
	Title, Heading, Description, CSRFToken, Role, Username string
	Rows                                                   []ClassRowViewModel
	AcademicYears                                          []ClassAcademicYearOption
	AcademicYearID, ClassID, Name, Error                   string
	Level, Page, Limit, PreviousPage, NextPage             int
	HasPrevious, HasNext, IsDetail                         bool
}

type ClassPageHandler struct {
	classes       service.ClassService
	academicYears service.AcademicYearService
	templates     *template.Template
}

func NewClassPageHandler(classes service.ClassService, academicYears service.AcademicYearService, templates *template.Template) (*ClassPageHandler, error) {
	if classes == nil || academicYears == nil || templates == nil {
		return nil, service.ErrInvalidDependency
	}
	return &ClassPageHandler{classes: classes, academicYears: academicYears, templates: templates}, nil
}

func (h *ClassPageHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	page, limit, err := parsePageOptions(r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	years, err := h.academicYears.List(r.Context(), service.ListAcademicYearsInput{Actor: actor, Options: service.ListOptions{Limit: maxAcademicYearPageSize}})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	yearOptions := make([]ClassAcademicYearOption, 0, len(years.AcademicYears))
	for _, year := range years.AcademicYears {
		if year != nil {
			yearOptions = append(yearOptions, ClassAcademicYearOption{ID: year.ID.String(), Name: year.Name})
		}
	}
	yearID := strings.TrimSpace(r.URL.Query().Get("academic_year_id"))
	vm := h.viewModel(actor, "Daftar Kelas", "Kelola kelas berdasarkan tahun ajaran.", yearOptions)
	vm.AcademicYearID, vm.Page, vm.Limit = yearID, page, limit
	if token, ok := middleware.CSRFToken(r); ok {
		vm.CSRFToken = token
	} else {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if yearID != "" {
		parsedID, parseErr := uuid.Parse(yearID)
		if parseErr != nil || parsedID == uuid.Nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		output, listErr := h.classes.List(r.Context(), service.ListClassesInput{Actor: actor, AcademicYearID: parsedID, Options: service.ListOptions{Limit: limit + 1, Offset: (page - 1) * limit}})
		if listErr != nil {
			httperror.Write(w, listErr)
			return
		}
		for _, class := range output.Classes {
			if class != nil {
				vm.Rows = append(vm.Rows, ClassRowViewModel{ID: class.ID.String(), Name: class.Name, Level: class.Level})
			}
		}
		vm.HasNext = len(vm.Rows) > limit
		if vm.HasNext {
			vm.Rows = vm.Rows[:limit]
		}
		vm.HasPrevious, vm.PreviousPage, vm.NextPage = page > 1, page-1, page+1
	}
	if err := h.render(w, vm); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *ClassPageHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if err := parseClassForm(w, r); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	name, level, err := parseClassValues(r)
	if err != nil {
		h.formError(w, r, actor, http.StatusBadRequest, "Nama kelas wajib diisi dan tingkat harus berupa angka 1 sampai 12.", "", name, level)
		return
	}
	output, err := h.classes.Create(r.Context(), service.CreateClassInput{Actor: actor, Name: name, Level: level})
	if err != nil {
		status := httperror.Status(err)
		if status >= 500 {
			httperror.Write(w, err)
			return
		}
		h.formError(w, r, actor, status, safeClassError(err), "", name, level)
		return
	}
	_ = output
	http.Redirect(w, r, "/classes", http.StatusSeeOther)
}

func (h *ClassPageHandler) Detail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil || id == uuid.Nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	output, err := h.classes.Get(r.Context(), service.GetClassInput{Actor: actor, ClassID: id})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	vm := h.viewModel(actor, "Detail Kelas", "Detail kelas.", nil)
	vm.IsDetail, vm.ClassID, vm.Name, vm.Level = true, output.Class.ID.String(), output.Class.Name, output.Class.Level
	if token, ok := middleware.CSRFToken(r); ok {
		vm.CSRFToken = token
	} else {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if err := h.render(w, vm); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *ClassPageHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil || id == uuid.Nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if err := parseClassForm(w, r); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	name, level, err := parseClassValues(r)
	if err != nil {
		h.formError(w, r, actor, http.StatusBadRequest, "Nama kelas wajib diisi dan tingkat harus berupa angka 1 sampai 12.", id.String(), name, level)
		return
	}
	_, err = h.classes.Update(r.Context(), service.UpdateClassInput{Actor: actor, ClassID: id, Name: name, Level: level})
	if err != nil {
		status := httperror.Status(err)
		if status >= 500 {
			httperror.Write(w, err)
			return
		}
		h.formError(w, r, actor, status, safeClassError(err), id.String(), name, level)
		return
	}
	http.Redirect(w, r, "/classes/"+id.String(), http.StatusSeeOther)
}

func (h *ClassPageHandler) viewModel(actor service.Actor, heading, description string, years []ClassAcademicYearOption) ClassPageViewModel {
	return ClassPageViewModel{Title: heading, Heading: heading, Description: description, Role: string(actor.Role), Username: actor.Username, AcademicYears: years, Page: 1, Limit: defaultAcademicYearPageSize}
}

func (h *ClassPageHandler) render(w http.ResponseWriter, vm ClassPageViewModel) error {
	// CSRF token is supplied by middleware on the request-specific view model in handlers.
	return h.templates.ExecuteTemplate(w, "classes", vm)
}

func (h *ClassPageHandler) formError(w http.ResponseWriter, r *http.Request, actor service.Actor, status int, message, id, name string, level int) {
	w.WriteHeader(status)
	vm := h.viewModel(actor, "Kelola Kelas", "Periksa kembali data kelas.", nil)
	vm.Error, vm.ClassID, vm.Name, vm.Level, vm.IsDetail = message, id, name, level, id != ""
	vm.CSRFToken = r.PostForm.Get("csrf_token")
	if vm.CSRFToken == "" {
		if token, ok := middleware.CSRFToken(r); ok {
			vm.CSRFToken = token
		}
	}
	_ = h.templates.ExecuteTemplate(w, "classes", vm)
}

func parseClassForm(w http.ResponseWriter, r *http.Request) error {
	if r.Header.Get("Content-Type") == "" {
		return errors.New("missing content type")
	}
	mediaType := strings.SplitN(r.Header.Get("Content-Type"), ";", 2)[0]
	if strings.TrimSpace(mediaType) != "application/x-www-form-urlencoded" {
		return errors.New("invalid content type")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxClassFormBytes)
	return r.ParseForm()
}

func parseClassValues(r *http.Request) (string, int, error) {
	name := strings.TrimSpace(r.PostForm.Get("name"))
	level, err := strconv.Atoi(strings.TrimSpace(r.PostForm.Get("level")))
	if name == "" || err != nil || level < 1 || level > 12 {
		return name, level, domain.ErrInvalidClassLevel
	}
	return name, level, nil
}

func safeClassError(err error) string {
	switch {
	case errors.Is(err, domain.ErrEmptyName):
		return "Nama kelas wajib diisi."
	case errors.Is(err, domain.ErrInvalidClassLevel):
		return "Tingkat kelas harus antara 1 sampai 12."
	case errors.Is(err, repository.ErrConflict):
		return "Nama kelas dengan tingkat tersebut sudah digunakan."
	case errors.Is(err, service.ErrForbidden):
		return "Akses ditolak untuk tindakan ini."
	default:
		return "Permintaan tidak dapat diproses."
	}
}

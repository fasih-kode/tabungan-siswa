package http

import (
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/httperror"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

const maxClassFormBytes = 16 << 10

type ClassRowViewModel struct {
	ID            string
	Name          string
	Level         int
	EffectiveDate string
}

type ClassAcademicYearOption struct {
	ID     string
	Name   string
	Status string
}

type ClassPageViewModel struct {
	Pagination                                                        PaginationViewModel
	Title, Heading, Description, CSRFToken, Role, Username, ActiveNav string
	Rows                                                              []ClassRowViewModel
	AcademicYears                                                     []ClassAcademicYearOption
	AcademicYearID, ClassID, Name, Error, EffectiveDate               string
	Level, Page, Limit, PreviousPage, NextPage                        int
	HasPrevious, HasNext, IsDetail                                    bool
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
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	page, limit, err := parsePageOptions(r)
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
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
			yearOptions = append(yearOptions, ClassAcademicYearOption{ID: year.ID.String(), Name: year.Name, Status: interfaceStatusLabel(string(year.Status))})
		}
	}
	yearID := strings.TrimSpace(r.URL.Query().Get("academic_year_id"))
	if yearID == "" {
		for _, year := range yearOptions {
			if year.Status == string(domain.AcademicYearOpen) {
				yearID = year.ID
				break
			}
		}
	}
	vm := h.viewModel(actor, "Daftar Kelas", "Kelola kelas berdasarkan tahun ajaran.", yearOptions)
	vm.AcademicYearID, vm.Page, vm.Limit = yearID, page, limit
	vm.Pagination = newPagination(r, page, 0, 0, false)
	if token, ok := middleware.CSRFToken(r); ok {
		vm.CSRFToken = token
	} else {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if yearID != "" {
		parsedID, parseErr := uuid.Parse(yearID)
		if parseErr != nil || parsedID == uuid.Nil {
			http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		output, listErr := h.classes.List(r.Context(), service.ListClassesInput{Actor: actor, AcademicYearID: parsedID, Options: service.ListOptions{Limit: limit + 1, Offset: (page - 1) * limit, IncludeTotal: true}})
		if listErr != nil {
			httperror.Write(w, listErr)
			return
		}
		for _, class := range output.Classes {
			if class != nil {
				vm.Rows = append(vm.Rows, ClassRowViewModel{ID: class.ID.String(), Name: class.Name, Level: class.Level, EffectiveDate: formatClassDate(class.EffectiveDate)})
			}
		}
		vm.HasNext = len(vm.Rows) > limit
		if vm.HasNext {
			vm.Rows = vm.Rows[:limit]
		}
		vm.HasPrevious, vm.PreviousPage, vm.NextPage = page > 1, page-1, page+1
		vm.Pagination = newPagination(r, page, output.Total, len(vm.Rows), vm.HasNext)
	}
	if redirectMissingPage(w, r, vm.Pagination) {
		return
	}
	if err := h.render(w, vm); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
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
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if err := parseClassForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	name, level, err := parseClassValues(r)
	if err != nil {
		h.formError(w, r, actor, http.StatusBadRequest, "Nama kelas wajib diisi dan tingkat harus berupa angka 1 sampai 12.", "", name, level)
		return
	}
	yearID := strings.TrimSpace(r.PostForm.Get("academic_year_id"))
	parsedYearID, err := uuid.Parse(yearID)
	if err != nil || parsedYearID == uuid.Nil {
		h.formError(w, r, actor, http.StatusBadRequest, "Pilih tahun ajaran.", "", name, level)
		return
	}
	years, err := h.academicYears.List(r.Context(), service.ListAcademicYearsInput{Actor: actor, Options: service.ListOptions{Limit: maxAcademicYearPageSize}})
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	validYear := false
	for _, year := range years.AcademicYears {
		if year != nil && year.ID == parsedYearID {
			validYear = true
			break
		}
	}
	if !validYear {
		h.formError(w, r, actor, http.StatusBadRequest, "Tahun ajaran yang dipilih tidak tersedia.", "", name, level)
		return
	}
	date, dateErr := time.Parse("2006-01-02", r.PostForm.Get("effective_date"))
	if dateErr != nil {
		h.formError(w, r, actor, http.StatusBadRequest, "Isi tanggal efektif penempatan kelas.", "", name, level)
		return
	}
	output, err := h.classes.Create(r.Context(), service.CreateClassInput{Actor: actor, Name: name, Level: level, AcademicYearID: parsedYearID, EffectiveDate: date})
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
	http.Redirect(w, r, "/classes?academic_year_id="+parsedYearID.String(), http.StatusSeeOther)
}

func (h *ClassPageHandler) Detail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil || id == uuid.Nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	output, err := h.classes.Get(r.Context(), service.GetClassInput{Actor: actor, ClassID: id})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	vm := h.viewModel(actor, "Detail Kelas", "Detail kelas.", nil)
	vm.IsDetail, vm.ClassID, vm.Name, vm.Level = true, output.Class.ID.String(), output.Class.Name, output.Class.Level
	vm.EffectiveDate = formatClassDate(output.Class.EffectiveDate)
	if output.Class.AcademicYearID != uuid.Nil {
		vm.AcademicYearID = output.Class.AcademicYearID.String()
	}
	years, err := h.academicYears.List(r.Context(), service.ListAcademicYearsInput{Actor: actor, Options: service.ListOptions{Limit: maxAcademicYearPageSize}})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	for _, year := range years.AcademicYears {
		if year != nil {
			vm.AcademicYears = append(vm.AcademicYears, ClassAcademicYearOption{ID: year.ID.String(), Name: year.Name})
		}
	}
	if token, ok := middleware.CSRFToken(r); ok {
		vm.CSRFToken = token
	} else {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if err := h.render(w, vm); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
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
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil || id == uuid.Nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if err := parseClassForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	name, level, err := parseClassValues(r)
	if err != nil {
		h.formError(w, r, actor, http.StatusBadRequest, "Nama kelas wajib diisi dan tingkat harus berupa angka 1 sampai 12.", id.String(), name, level)
		return
	}
	yearID, yearErr := uuid.Parse(r.PostForm.Get("academic_year_id"))
	date, dateErr := time.Parse("2006-01-02", r.PostForm.Get("effective_date"))
	if yearErr != nil || yearID == uuid.Nil || dateErr != nil {
		h.formError(w, r, actor, http.StatusBadRequest, "Pilih tahun ajaran dan isi tanggal efektif kelas.", id.String(), name, level)
		return
	}
	_, err = h.classes.Update(r.Context(), service.UpdateClassInput{Actor: actor, ClassID: id, Name: name, Level: level, AcademicYearID: yearID, EffectiveDate: date})
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
	return ClassPageViewModel{Title: heading, Heading: heading, Description: description, Role: string(actor.Role), Username: actor.Username, ActiveNav: "classes", AcademicYears: years, Page: 1, Limit: defaultAcademicYearPageSize}
}

func (h *ClassPageHandler) render(w http.ResponseWriter, vm ClassPageViewModel) error {
	// CSRF token is supplied by middleware on the request-specific view model in handlers.
	return h.templates.ExecuteTemplate(w, "classes", vm)
}

func (h *ClassPageHandler) formError(w http.ResponseWriter, r *http.Request, actor service.Actor, status int, message, id, name string, level int) {
	w.WriteHeader(status)
	var yearOptions []ClassAcademicYearOption
	if years, err := h.academicYears.List(r.Context(), service.ListAcademicYearsInput{Actor: actor, Options: service.ListOptions{Limit: maxAcademicYearPageSize}}); err == nil {
		for _, year := range years.AcademicYears {
			if year != nil {
				yearOptions = append(yearOptions, ClassAcademicYearOption{ID: year.ID.String(), Name: year.Name, Status: interfaceStatusLabel(string(year.Status))})
			}
		}
	}
	vm := h.viewModel(actor, "Daftar Kelas", "Kelola kelas berdasarkan tahun ajaran.", yearOptions)
	vm.Error, vm.ClassID, vm.Name, vm.Level, vm.IsDetail = message, id, name, level, id != ""
	vm.AcademicYearID = r.PostForm.Get("academic_year_id")
	vm.EffectiveDate = r.PostForm.Get("effective_date")
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
	case errors.Is(err, domain.ErrInvalidDateRange):
		return "Tanggal efektif harus berada di dalam periode tahun ajaran."
	case errors.Is(err, domain.ErrInvalidTransition):
		return "Tahun ajaran kelas yang sudah tersimpan tidak dapat diganti."
	case errors.Is(err, repository.ErrConflict):
		return "Nama kelas dengan tingkat tersebut sudah digunakan."
	case errors.Is(err, service.ErrForbidden):
		return "Akses ditolak untuk tindakan ini."
	default:
		return "Permintaan tidak dapat diproses."
	}
}

func formatClassDate(date time.Time) string {
	if date.IsZero() {
		return ""
	}
	return date.Format("2006-01-02")
}

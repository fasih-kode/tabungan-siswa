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

const (
	defaultAcademicYearPageSize = 20
	maxAcademicYearPageSize     = 100
	maxAcademicYearFormBytes    = 16 << 10
)

type AcademicYearRowViewModel struct {
	ID        string
	Name      string
	StartDate string
	EndDate   string
	Status    string
	CanClose  bool
	CanReopen bool
}

type AcademicYearPageViewModel struct {
	Title        string
	Heading      string
	Description  string
	CSRFToken    string
	Role         string
	Username     string
	Rows         []AcademicYearRowViewModel
	AcademicYear *AcademicYearRowViewModel
	Name         string
	StartDate    string
	EndDate      string
	Error        string
	Page         int
	Limit        int
	PreviousPage int
	NextPage     int
	HasPrevious  bool
	HasNext      bool
}

type AcademicYearPageHandler struct {
	academicYears service.AcademicYearService
	templates     *template.Template
}

func NewAcademicYearPageHandler(
	academicYears service.AcademicYearService,
	templates *template.Template,
) (*AcademicYearPageHandler, error) {
	if academicYears == nil || templates == nil {
		return nil, service.ErrInvalidDependency
	}
	return &AcademicYearPageHandler{academicYears: academicYears, templates: templates}, nil
}

func (h *AcademicYearPageHandler) List(w http.ResponseWriter, r *http.Request) {
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
	output, err := h.academicYears.List(r.Context(), service.ListAcademicYearsInput{
		Actor: actor, Options: service.ListOptions{Limit: limit + 1, Offset: (page - 1) * limit},
	})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	csrfToken, ok := middleware.CSRFToken(r)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	rows := make([]AcademicYearRowViewModel, 0, len(output.AcademicYears))
	for _, year := range output.AcademicYears {
		if year == nil {
			continue
		}
		rows = append(rows, academicYearRow(*year))
	}
	hasNext := len(rows) > limit
	if hasNext {
		rows = rows[:limit]
	}
	vm := AcademicYearPageViewModel{
		Title: "Tahun Ajaran", Heading: "Tahun Ajaran",
		Description: "Kelola periode dan status tahun ajaran.", CSRFToken: csrfToken,
		Role: string(actor.Role), Username: actor.Username, Rows: rows, Page: page, Limit: limit,
		HasPrevious: page > 1, HasNext: hasNext, PreviousPage: page - 1, NextPage: page + 1,
	}
	if err := h.templates.ExecuteTemplate(w, "academic-years", vm); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *AcademicYearPageHandler) Create(w http.ResponseWriter, r *http.Request) {
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
	if err := parseAcademicYearForm(w, r); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	startDate, startErr := time.Parse("2006-01-02", r.PostForm.Get("start_date"))
	endDate, endErr := time.Parse("2006-01-02", r.PostForm.Get("end_date"))
	if startErr != nil || endErr != nil || strings.TrimSpace(r.PostForm.Get("name")) == "" {
		h.renderFormError(w, actor, http.StatusBadRequest, "Nama dan tanggal tahun ajaran wajib diisi dengan format yang benar.", r.PostForm.Get("csrf_token"), r.PostForm.Get("name"), r.PostForm.Get("start_date"), r.PostForm.Get("end_date"))
		return
	}
	_, err := h.academicYears.Create(r.Context(), service.CreateAcademicYearInput{
		Actor: actor, Name: r.PostForm.Get("name"), StartDate: startDate, EndDate: endDate,
	})
	if err != nil {
		status := httperror.Status(err)
		if status >= 500 {
			httperror.Write(w, err)
			return
		}
		h.renderFormError(w, actor, status, safeAcademicYearError(err), r.PostForm.Get("csrf_token"), r.PostForm.Get("name"), r.PostForm.Get("start_date"), r.PostForm.Get("end_date"))
		return
	}
	http.Redirect(w, r, "/academic-years", http.StatusSeeOther)
}

func (h *AcademicYearPageHandler) Detail(w http.ResponseWriter, r *http.Request) {
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
	output, err := h.academicYears.Get(r.Context(), service.GetAcademicYearInput{Actor: actor, AcademicYearID: id})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	csrfToken, ok := middleware.CSRFToken(r)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	row := academicYearRow(*output.AcademicYear)
	vm := AcademicYearPageViewModel{Title: "Detail Tahun Ajaran", Heading: row.Name, Description: "Detail dan lifecycle tahun ajaran.", CSRFToken: csrfToken, Role: string(actor.Role), Username: actor.Username, AcademicYear: &row}
	if err := h.templates.ExecuteTemplate(w, "academic-years", vm); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *AcademicYearPageHandler) Transition(w http.ResponseWriter, r *http.Request) {
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
	if err := parseAcademicYearForm(w, r); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	var transitionErr error
	switch r.PathValue("action") {
	case "start-closing":
		_, transitionErr = h.academicYears.StartClosing(r.Context(), service.StartClosingAcademicYearInput{Actor: actor, AcademicYearID: id})
	case "close":
		_, transitionErr = h.academicYears.Close(r.Context(), service.CloseAcademicYearInput{Actor: actor, AcademicYearID: id})
	case "reopen":
		_, transitionErr = h.academicYears.Reopen(r.Context(), service.ReopenAcademicYearInput{Actor: actor, AcademicYearID: id})
	default:
		http.NotFound(w, r)
		return
	}
	if transitionErr != nil {
		httperror.Write(w, transitionErr)
		return
	}
	http.Redirect(w, r, "/academic-years/"+id.String(), http.StatusSeeOther)
}

func (h *AcademicYearPageHandler) renderFormError(
	w http.ResponseWriter,
	actor service.Actor,
	status int,
	message string,
	csrfToken string,
	name string,
	startDate string,
	endDate string,
) {
	w.WriteHeader(status)
	vm := AcademicYearPageViewModel{
		Title: "Tahun Ajaran", Heading: "Tahun Ajaran",
		Description: "Kelola periode dan status tahun ajaran.",
		CSRFToken:   csrfToken, Role: string(actor.Role), Username: actor.Username, Name: name,
		StartDate: startDate, EndDate: endDate, Error: message,
	}
	if err := h.templates.ExecuteTemplate(w, "academic-years", vm); err != nil {
		return
	}
}

func academicYearRow(year domain.AcademicYear) AcademicYearRowViewModel {
	return AcademicYearRowViewModel{
		ID: year.ID.String(), Name: year.Name,
		StartDate: year.StartDate.Format("2006-01-02"), EndDate: year.EndDate.Format("2006-01-02"),
		Status: string(year.Status), CanClose: year.Status == domain.AcademicYearClosing,
		CanReopen: year.Status == domain.AcademicYearClosed,
	}
}

func parsePageOptions(r *http.Request) (int, int, error) {
	page, limit := 1, defaultAcademicYearPageSize
	if raw := r.URL.Query().Get("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return 0, 0, errors.New("invalid page")
		}
		page = value
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > maxAcademicYearPageSize {
			return 0, 0, errors.New("invalid limit")
		}
		limit = value
	}
	if page-1 > int(^uint(0)>>1)/limit {
		return 0, 0, errors.New("page offset overflow")
	}
	return page, limit, nil
}

func limitAcademicYearRequestBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxAcademicYearFormBytes)
		next.ServeHTTP(w, r)
	})
}

func parseAcademicYearForm(w http.ResponseWriter, r *http.Request) error {
	if r.Header.Get("Content-Type") == "" {
		return errors.New("missing content type")
	}
	mediaType := strings.SplitN(r.Header.Get("Content-Type"), ";", 2)[0]
	if strings.TrimSpace(mediaType) != "application/x-www-form-urlencoded" {
		return errors.New("invalid content type")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAcademicYearFormBytes)
	return r.ParseForm()
}

func safeAcademicYearError(err error) string {
	switch {
	case errors.Is(err, domain.ErrEmptyName):
		return "Nama tahun ajaran wajib diisi."
	case errors.Is(err, domain.ErrInvalidDateRange):
		return "Tanggal mulai harus lebih awal daripada tanggal akhir."
	case errors.Is(err, repository.ErrConflict), errors.Is(err, domain.ErrInvalidTransition):
		return "Perubahan tidak dapat dilakukan karena bertentangan dengan status atau data yang ada."
	case errors.Is(err, service.ErrForbidden):
		return "Akses ditolak untuk tindakan ini."
	default:
		return "Permintaan tidak dapat diproses."
	}
}

func methodNotAllowed(w http.ResponseWriter, allowedMethods string) {
	w.Header().Set("Allow", allowedMethods)
	http.Error(
		w,
		http.StatusText(http.StatusMethodNotAllowed),
		http.StatusMethodNotAllowed,
	)
}

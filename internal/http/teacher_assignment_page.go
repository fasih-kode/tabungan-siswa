package http

import (
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/httperror"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

const maxTeacherAssignmentFormBytes = 16 << 10

type TeacherAssignmentRow struct {
	ClassID, ClassName, TeacherID, TeacherUsername, AssignmentID string
	Level                                                        int
	Assigned                                                     bool
}

type TeacherAssignmentOption struct{ ID, Name string }

type TeacherAssignmentPageViewModel struct {
	Title, Heading, Description, CSRFToken, Role, Username string
	AcademicYearID, TeacherID, ClassID, Error              string
	AcademicYears                                          []TeacherAssignmentOption
	Teachers                                               []TeacherAssignmentOption
	Rows                                                   []TeacherAssignmentRow
}

type TeacherAssignmentPageHandler struct {
	assignments service.ClassAssignmentService
	users       service.UserService
	classes     service.ClassService
	years       service.AcademicYearService
	templates   *template.Template
}

func NewTeacherAssignmentPageHandler(assignments service.ClassAssignmentService, users service.UserService, classes service.ClassService, years service.AcademicYearService, templates *template.Template) (*TeacherAssignmentPageHandler, error) {
	if assignments == nil || users == nil || classes == nil || years == nil || templates == nil {
		return nil, service.ErrInvalidDependency
	}
	return &TeacherAssignmentPageHandler{assignments: assignments, users: users, classes: classes, years: years, templates: templates}, nil
}

func (h *TeacherAssignmentPageHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if actor.Role != domain.RoleAdmin {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	vm := h.baseVM(actor)
	if token, ok := middleware.CSRFToken(r); ok {
		vm.CSRFToken = token
	} else {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	years, err := h.years.List(r.Context(), service.ListAcademicYearsInput{Actor: actor, Options: service.ListOptions{Limit: maxAcademicYearPageSize}})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	for _, year := range years.AcademicYears {
		if year != nil {
			vm.AcademicYears = append(vm.AcademicYears, TeacherAssignmentOption{ID: year.ID.String(), Name: year.Name})
		}
	}
	teachers, err := h.users.ListByRole(r.Context(), service.ListUsersByRoleInput{Actor: actor, Role: domain.RoleWaliKelas, Options: service.ListOptions{Limit: maxAcademicYearPageSize}})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	for _, teacher := range teachers.Users {
		if teacher != nil {
			vm.Teachers = append(vm.Teachers, TeacherAssignmentOption{ID: teacher.ID.String(), Name: teacher.Username})
		}
	}
	vm.AcademicYearID = strings.TrimSpace(r.URL.Query().Get("academic_year_id"))
	if vm.AcademicYearID != "" {
		yearID, parseErr := uuid.Parse(vm.AcademicYearID)
		if parseErr != nil || yearID == uuid.Nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		classes, listErr := h.classes.List(r.Context(), service.ListClassesInput{Actor: actor, AcademicYearID: yearID, Options: service.ListOptions{Limit: maxAcademicYearPageSize}})
		if listErr != nil {
			httperror.Write(w, listErr)
			return
		}
		teacherNames := make(map[uuid.UUID]string, len(teachers.Users))
		for _, teacher := range teachers.Users {
			if teacher != nil {
				teacherNames[teacher.ID] = teacher.Username
			}
		}
		for _, class := range classes.Classes {
			if class == nil {
				continue
			}
			row := TeacherAssignmentRow{ClassID: class.ID.String(), ClassName: class.Name, Level: class.Level}
			current, currentErr := h.assignments.GetCurrentByClass(r.Context(), service.GetCurrentClassAssignmentInput{Actor: actor, ClassID: class.ID, AcademicYearID: yearID, Date: classAssignmentDateForYear(years.AcademicYears, yearID)})
			if currentErr == nil && current.Assignment != nil {
				row.Assigned = true
				row.AssignmentID = current.Assignment.ID.String()
				row.TeacherID = current.Assignment.UserID.String()
				row.TeacherUsername = teacherNames[current.Assignment.UserID]
			}
			if currentErr != nil && !errors.Is(currentErr, repository.ErrNotFound) {
				httperror.Write(w, currentErr)
				return
			}
			vm.Rows = append(vm.Rows, row)
		}
	}
	if err := h.templates.ExecuteTemplate(w, "teacher-assignments", vm); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *TeacherAssignmentPageHandler) Create(w http.ResponseWriter, r *http.Request) {
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
	if actor.Role != domain.RoleAdmin {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	if err := parseTeacherAssignmentForm(w, r); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	userID, userErr := uuid.Parse(strings.TrimSpace(r.PostForm.Get("teacher_id")))
	classID, classErr := uuid.Parse(strings.TrimSpace(r.PostForm.Get("class_id")))
	yearID, yearErr := uuid.Parse(strings.TrimSpace(r.PostForm.Get("academic_year_id")))
	if userErr != nil || classErr != nil || yearErr != nil || userID == uuid.Nil || classID == uuid.Nil || yearID == uuid.Nil {
		h.formError(w, r, actor, http.StatusBadRequest, "Pilih tahun ajaran, kelas, dan wali kelas yang valid.")
		return
	}
	_, err := h.assignments.Assign(r.Context(), service.AssignTeacherInput{Actor: actor, UserID: userID, ClassID: classID, AcademicYearID: yearID})
	if err != nil {
		status := httperror.Status(err)
		if status >= 500 {
			httperror.Write(w, err)
			return
		}
		message := "Penugasan tidak dapat diproses. Periksa kembali pilihan dan periode penugasan."
		if errors.Is(err, service.ErrForbidden) {
			message = "Akses ditolak untuk tindakan ini."
		}
		h.formError(w, r, actor, status, message)
		return
	}
	http.Redirect(w, r, "/teacher-assignments?academic_year_id="+url.QueryEscape(yearID.String()), http.StatusSeeOther)
}

func (h *TeacherAssignmentPageHandler) baseVM(actor service.Actor) TeacherAssignmentPageViewModel {
	return TeacherAssignmentPageViewModel{Title: "Penugasan Wali Kelas", Heading: "Penugasan Wali Kelas", Description: "Kelola penugasan wali kelas berdasarkan tahun ajaran.", Role: string(actor.Role), Username: actor.Username}
}

func (h *TeacherAssignmentPageHandler) formError(w http.ResponseWriter, r *http.Request, actor service.Actor, status int, message string) {
	vm := h.baseVM(actor)
	vm.Error = message
	vm.AcademicYearID = r.PostForm.Get("academic_year_id")
	vm.ClassID = r.PostForm.Get("class_id")
	vm.TeacherID = r.PostForm.Get("teacher_id")
	if token, ok := middleware.CSRFToken(r); ok {
		vm.CSRFToken = token
	}
	// Reload options so the form remains usable after a recoverable error.
	years, err := h.years.List(r.Context(), service.ListAcademicYearsInput{Actor: actor, Options: service.ListOptions{Limit: maxAcademicYearPageSize}})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	for _, y := range years.AcademicYears {
		if y != nil {
			vm.AcademicYears = append(vm.AcademicYears, TeacherAssignmentOption{ID: y.ID.String(), Name: y.Name})
		}
	}
	users, err := h.users.ListByRole(r.Context(), service.ListUsersByRoleInput{Actor: actor, Role: domain.RoleWaliKelas, Options: service.ListOptions{Limit: maxAcademicYearPageSize}})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	for _, u := range users.Users {
		if u != nil {
			vm.Teachers = append(vm.Teachers, TeacherAssignmentOption{ID: u.ID.String(), Name: u.Username})
		}
	}
	if yearID, parseErr := uuid.Parse(vm.AcademicYearID); parseErr == nil && yearID != uuid.Nil {
		classes, classErr := h.classes.List(r.Context(), service.ListClassesInput{Actor: actor, AcademicYearID: yearID, Options: service.ListOptions{Limit: maxAcademicYearPageSize}})
		if classErr != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		for _, c := range classes.Classes {
			if c != nil {
				vm.Rows = append(vm.Rows, TeacherAssignmentRow{ClassID: c.ID.String(), ClassName: c.Name, Level: c.Level})
			}
		}
	}
	w.WriteHeader(status)
	if err := h.templates.ExecuteTemplate(w, "teacher-assignments", vm); err != nil {
		// The status may already be committed; do not leak template internals.
		return
	}
}

func parseTeacherAssignmentForm(w http.ResponseWriter, r *http.Request) error {
	contentType := strings.TrimSpace(strings.SplitN(r.Header.Get("Content-Type"), ";", 2)[0])
	if contentType != "application/x-www-form-urlencoded" {
		return errors.New("invalid content type")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxTeacherAssignmentFormBytes)
	return r.ParseForm()
}

func classAssignmentDateForYear(years []*domain.AcademicYear, id uuid.UUID) time.Time {
	for _, year := range years {
		if year != nil && year.ID == id {
			return year.StartDate
		}
	}
	return time.Time{}
}

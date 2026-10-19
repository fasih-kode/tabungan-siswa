package http

import (
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/httperror"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

const (
	maxManagedUsers    = 100
	maxStudentsForLink = 5000
)

type ManagedUserRow struct {
	ID, Username, Role, StudentName string
}

type UnlinkedStudentOption struct {
	ID, Label string
}

type UserClassOption struct {
	ID, Label string
}

type UserPageViewModel struct {
	Pagination                                                                                 PaginationViewModel
	Title, Heading, Description, CSRFToken, Role, Username, ActiveNav                          string
	RoleFilter, FullName, StudentID, ClassID, OpenAcademicYear, Error, Notice, CreatedUsername string
	TemporaryPassword                                                                          string
	Users                                                                                      []ManagedUserRow
	Students                                                                                   []UnlinkedStudentOption
	Classes                                                                                    []UserClassOption
	Page, Limit, PreviousPage, NextPage                                                        int
	HasPrevious, HasNext                                                                       bool
	PasswordChangeRequired                                                                     bool
}

type UserPageHandler struct {
	users             service.UserService
	students          service.StudentService
	academicYears     service.AcademicYearService
	classes           service.ClassService
	passwordHasher    security.PasswordHasher
	templates         *template.Template
	credentialReveals *credentialRevealStore
}

func NewUserPageHandler(users service.UserService, students service.StudentService, academicYears service.AcademicYearService, classes service.ClassService, passwordHasher security.PasswordHasher, templates *template.Template) (*UserPageHandler, error) {
	if users == nil || students == nil || academicYears == nil || classes == nil || passwordHasher == nil || templates == nil {
		return nil, service.ErrInvalidDependency
	}
	return &UserPageHandler{users: users, students: students, academicYears: academicYears, classes: classes, passwordHasher: passwordHasher, templates: templates, credentialReveals: newCredentialRevealStore()}, nil
}

func (h *UserPageHandler) StudentOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
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
	classID, err := uuid.Parse(r.URL.Query().Get("class_id"))
	if err != nil || classID == uuid.Nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	year, err := h.openAcademicYear(r, actor)
	if err != nil {
		http.Error(w, http.StatusText(httperror.Status(err)), httperror.Status(err))
		return
	}
	if year == nil {
		http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
		return
	}
	classOutput, err := h.classes.Get(r.Context(), service.GetClassInput{Actor: actor, ClassID: classID})
	if err != nil {
		http.Error(w, http.StatusText(httperror.Status(err)), httperror.Status(err))
		return
	}
	if classOutput.Class == nil || classOutput.Class.AcademicYearID != year.ID {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	output, err := h.students.List(r.Context(), service.ListStudentsInput{
		Actor: actor, IncludePlacement: true, AcademicYearID: year.ID, ClassID: classID,
		Options: service.ListOptions{Limit: maxStudentsForLink, StudentStatus: string(domain.StudentActive)},
	})
	if err != nil {
		http.Error(w, http.StatusText(httperror.Status(err)), httperror.Status(err))
		return
	}
	options := make([]UnlinkedStudentOption, 0, len(output.Students))
	for _, student := range output.Students {
		if student == nil || !student.IsActive() || student.UserID != nil || student.NIS == nil || output.Placements[student.ID].ClassID != classID {
			continue
		}
		options = append(options, UnlinkedStudentOption{ID: student.ID.String(), Label: student.Name + " — " + *student.NIS})
	}
	if err := json.NewEncoder(w).Encode(options); err != nil {
		return
	}
}

func (h *UserPageHandler) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
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
	page, limit, err := parsePageOptions(r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if limit >= maxManagedUsers {
		limit = maxManagedUsers - 1
	}
	role := normalizeManagedRole(r.URL.Query().Get("role"))
	vm, err := h.load(r, actor, role, page, limit)
	if err != nil {
		http.Error(w, http.StatusText(httperror.Status(err)), httperror.Status(err))
		return
	}
	if token := r.URL.Query().Get("credential"); token != "" {
		if reveal, ok := h.credentialReveals.Take(actor.UserID, token); ok {
			vm.CreatedUsername = reveal.Username
			vm.TemporaryPassword = reveal.Password
			vm.PasswordChangeRequired = reveal.PasswordChangeRequired
			vm.Notice = "Salin password sementara ini sekarang. Password hanya ditampilkan satu kali."
		}
	} else if r.URL.Query().Get("student_created") == "1" {
		vm.CreatedUsername = r.URL.Query().Get("username")
		vm.Notice = "Akun siswa berhasil dibuat. Password sementara ditampilkan satu kali."
	}
	if redirectMissingPage(w, r, vm.Pagination) {
		return
	}
	if err := h.templates.ExecuteTemplate(w, "users", vm); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *UserPageHandler) Create(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
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
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	role := normalizeManagedRole(r.PostForm.Get("role"))
	fullName := strings.TrimSpace(r.PostForm.Get("full_name"))
	studentID := strings.TrimSpace(r.PostForm.Get("student_id"))
	classIDValue := strings.TrimSpace(r.PostForm.Get("class_id"))
	if role != string(domain.RoleWaliKelas) && role != string(domain.RoleSiswa) {
		h.formError(w, r, actor, role, fullName, studentID, http.StatusBadRequest, "Pilih role wali kelas atau siswa.")
		return
	}
	baseUsername := ""
	var student *domain.Student
	if role == string(domain.RoleWaliKelas) {
		baseUsername = usernameFromName(fullName)
		if fullName == "" || baseUsername == "" {
			h.formError(w, r, actor, role, fullName, studentID, http.StatusBadRequest, "Nama wali kelas wajib diisi.")
			return
		}
	} else {
		id, err := uuid.Parse(studentID)
		if err != nil || id == uuid.Nil {
			h.formError(w, r, actor, role, fullName, studentID, http.StatusBadRequest, "Pilih data siswa yang belum memiliki akun.")
			return
		}
		classID, err := uuid.Parse(classIDValue)
		if err != nil || classID == uuid.Nil {
			h.formError(w, r, actor, role, fullName, studentID, http.StatusBadRequest, "Pilih kelas pada tahun ajaran OPEN.")
			return
		}
		year, err := h.openAcademicYear(r, actor)
		if err != nil {
			h.renderFormError(w, r, actor, role, fullName, studentID, err)
			return
		}
		if year == nil {
			h.formError(w, r, actor, role, fullName, studentID, http.StatusConflict, "Belum ada tahun ajaran berstatus OPEN.")
			return
		}
		class, err := h.classes.Get(r.Context(), service.GetClassInput{Actor: actor, ClassID: classID})
		if err != nil || class.Class == nil || class.Class.AcademicYearID != year.ID {
			h.formError(w, r, actor, role, fullName, studentID, http.StatusBadRequest, "Kelas harus berasal dari tahun ajaran OPEN.")
			return
		}
		placement, err := h.students.List(r.Context(), service.ListStudentsInput{Actor: actor, IncludePlacement: true, AcademicYearID: year.ID, ClassID: classID, Options: service.ListOptions{Limit: maxStudentsForLink, StudentStatus: string(domain.StudentActive)}})
		if err != nil {
			h.renderFormError(w, r, actor, role, fullName, studentID, err)
			return
		}
		for _, candidate := range placement.Students {
			if candidate != nil && candidate.ID == id {
				student = candidate
				break
			}
		}
		if student == nil {
			h.formError(w, r, actor, role, fullName, studentID, http.StatusConflict, "Siswa harus aktif dan terdaftar pada kelas yang dipilih di tahun ajaran OPEN.")
			return
		}
		if !student.IsActive() || student.UserID != nil || student.NIS == nil {
			h.formError(w, r, actor, role, fullName, studentID, http.StatusConflict, "Siswa tidak aktif, sudah memiliki akun, atau belum memiliki NIS.")
			return
		}
		baseUsername = usernameFromName(*student.NIS)
		if baseUsername == "" {
			h.formError(w, r, actor, role, fullName, studentID, http.StatusBadRequest, "NIS siswa tidak dapat digunakan sebagai username.")
			return
		}
	}
	username, err := h.availableUsername(r, actor, baseUsername)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	temporaryPassword, err := generateTemporaryPassword()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	passwordHash, err := h.passwordHasher.Hash(temporaryPassword)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	passwordChangeRequired := role == string(domain.RoleWaliKelas)
	credentialToken, err := h.credentialReveals.Put(actor.UserID, username, temporaryPassword, passwordChangeRequired)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if role == string(domain.RoleWaliKelas) {
		_, err = h.users.Create(r.Context(), service.CreateUserInput{Actor: actor, Username: username, PasswordHash: passwordHash, Role: domain.RoleWaliKelas, PasswordChangeRequired: true})
	} else {
		_, err = h.users.CreateStudentAccount(r.Context(), service.CreateStudentAccountInput{Actor: actor, StudentID: student.ID, Username: username, PasswordHash: passwordHash})
	}
	if err != nil {
		h.renderFormError(w, r, actor, role, fullName, studentID, err)
		return
	}
	if role == string(domain.RoleSiswa) {
		http.Redirect(w, r, "/users?role="+urlQueryEscape(role)+"&student_created=1&credential="+urlQueryEscape(credentialToken), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/users?role="+urlQueryEscape(role)+"&credential="+urlQueryEscape(credentialToken), http.StatusSeeOther)
}

func (h *UserPageHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
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
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil || id == uuid.Nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	output, err := h.users.Get(r.Context(), service.GetUserInput{Actor: actor, UserID: id})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	if output.User.Role != domain.RoleWaliKelas && output.User.Role != domain.RoleSiswa {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	temporaryPassword, err := generateTemporaryPassword()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	passwordHash, err := h.passwordHasher.Hash(temporaryPassword)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	passwordChangeRequired := output.User.Role == domain.RoleWaliKelas
	credentialToken, err := h.credentialReveals.Put(actor.UserID, output.User.Username, temporaryPassword, passwordChangeRequired)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	_, err = h.users.Update(r.Context(), service.UpdateUserInput{Actor: actor, UserID: id, Username: output.User.Username, Role: output.User.Role, PasswordHash: passwordHash, PasswordChangeRequired: output.User.Role == domain.RoleWaliKelas})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	http.Redirect(w, r, "/users?role="+urlQueryEscape(string(output.User.Role))+"&credential="+urlQueryEscape(credentialToken), http.StatusSeeOther)
}

func (h *UserPageHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		h.renderPasswordPage(w, r, actor, "", r.URL.Query().Get("updated") == "1")
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w, "GET, HEAD, POST")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	current, next, confirm := r.PostForm.Get("current_password"), r.PostForm.Get("new_password"), r.PostForm.Get("confirm_password")
	if current == "" || len([]byte(next)) < 12 || len([]byte(next)) > security.MaxPasswordBytes || next != confirm || next == current {
		h.renderPasswordPage(w, r, actor, "Password baru harus berbeda dari password sementara, minimal 12 karakter, dan cocok dengan konfirmasi.", false)
		return
	}
	passwordHash, err := h.passwordHasher.Hash(next)
	if err != nil {
		h.renderPasswordPage(w, r, actor, "Kata sandi baru tidak dapat digunakan.", false)
		return
	}
	_, err = h.users.Update(r.Context(), service.UpdateUserInput{Actor: actor, UserID: actor.UserID, Username: actor.Username, Role: actor.Role, PasswordHash: passwordHash, CurrentPassword: current})
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			h.renderPasswordPage(w, r, actor, "Kata sandi saat ini tidak sesuai.", false)
			return
		}
		if httperror.Status(err) < http.StatusInternalServerError {
			h.renderPasswordPage(w, r, actor, "Kata sandi tidak dapat diperbarui.", false)
			return
		}
		log.Printf("change password update failed: %v", err)
		h.renderPasswordPage(w, r, actor, "Password belum berhasil diperbarui. Silakan coba lagi.", false)
		return
	}
	http.Redirect(w, r, "/account/password?updated=1", http.StatusSeeOther)
}

func (h *UserPageHandler) load(r *http.Request, actor service.Actor, role string, page, limit int) (UserPageViewModel, error) {
	if role != string(domain.RoleWaliKelas) && role != string(domain.RoleSiswa) {
		role = string(domain.RoleWaliKelas)
	}
	vm := UserPageViewModel{Title: "Pengguna", Heading: "Pengguna", Description: "Kelola akun wali kelas dan siswa.", Role: string(actor.Role), Username: actor.Username, ActiveNav: "users", RoleFilter: role, Page: page, Limit: limit}
	if token, ok := middleware.CSRFToken(r); ok {
		vm.CSRFToken = token
	} else {
		return vm, service.ErrInvalidCredentials
	}
	users, err := h.users.ListByRole(r.Context(), service.ListUsersByRoleInput{Actor: actor, Role: domain.UserRole(role), Options: service.ListOptions{Limit: limit + 1, Offset: (page - 1) * limit, IncludeTotal: true}})
	if err != nil {
		return vm, err
	}
	vm.HasNext = len(users.Users) > limit
	if len(users.Users) > limit {
		users.Users = users.Users[:limit]
	}
	for _, user := range users.Users {
		if user != nil {
			vm.Users = append(vm.Users, ManagedUserRow{ID: user.ID.String(), Username: user.Username, Role: string(user.Role)})
		}
	}
	vm.HasPrevious = page > 1
	vm.PreviousPage = page - 1
	vm.NextPage = page + 1
	vm.Pagination = newPagination(r, page, users.Total, len(vm.Users), vm.HasNext)
	year, err := h.openAcademicYear(r, actor)
	if err != nil {
		return vm, err
	}
	if year != nil {
		vm.OpenAcademicYear = year.Name
		classes, err := h.classes.List(r.Context(), service.ListClassesInput{Actor: actor, AcademicYearID: year.ID, Options: service.ListOptions{Limit: maxStudentsForLink}})
		if err != nil {
			return vm, err
		}
		for _, class := range classes.Classes {
			if class != nil {
				vm.Classes = append(vm.Classes, UserClassOption{ID: class.ID.String(), Label: class.Name})
			}
		}
	}
	if role == string(domain.RoleSiswa) {
		vm.ClassID = r.URL.Query().Get("class_id")
		students, err := h.students.List(r.Context(), service.ListStudentsInput{Actor: actor, Options: service.ListOptions{Limit: maxStudentsForLink}})
		if err != nil {
			return vm, err
		}
		studentByUser := make(map[uuid.UUID]string, len(students.Students))
		for _, student := range students.Students {
			if student != nil && student.UserID != nil {
				studentByUser[*student.UserID] = student.Name
			}
		}
		for i := range vm.Users {
			id, _ := uuid.Parse(vm.Users[i].ID)
			vm.Users[i].StudentName = studentByUser[id]
		}
	}
	return vm, nil
}

func (h *UserPageHandler) availableUsername(r *http.Request, actor service.Actor, base string) (string, error) {
	for suffix := 1; ; suffix++ {
		candidate := base
		if suffix > 1 {
			candidate += strconv.Itoa(suffix)
		}
		_, err := h.users.GetByUsername(r.Context(), service.GetUserByUsernameInput{Actor: actor, Username: candidate})
		if errors.Is(err, repository.ErrNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
	}
}

func (h *UserPageHandler) openAcademicYear(r *http.Request, actor service.Actor) (*domain.AcademicYear, error) {
	output, err := h.academicYears.List(r.Context(), service.ListAcademicYearsInput{Actor: actor, Options: service.ListOptions{Limit: maxStudentsForLink}})
	if err != nil {
		return nil, err
	}
	var open *domain.AcademicYear
	for _, year := range output.AcademicYears {
		if year != nil && year.IsOpen() && (open == nil || year.StartDate.After(open.StartDate)) {
			open = year
		}
	}
	return open, nil
}

func (h *UserPageHandler) formError(w http.ResponseWriter, r *http.Request, actor service.Actor, role, fullName, studentID string, status int, message string) {
	vm, err := h.load(r, actor, role, 1, 20)
	if err != nil {
		http.Error(w, http.StatusText(httperror.Status(err)), httperror.Status(err))
		return
	}
	vm.Error, vm.FullName, vm.StudentID = message, fullName, studentID
	vm.ClassID = r.PostForm.Get("class_id")
	w.WriteHeader(status)
	_ = h.templates.ExecuteTemplate(w, "users", vm)
}

func (h *UserPageHandler) renderFormError(w http.ResponseWriter, r *http.Request, actor service.Actor, role, fullName, studentID string, err error) {
	status := httperror.Status(err)
	if status >= http.StatusInternalServerError {
		httperror.Write(w, err)
		return
	}
	message := "Akun tidak dapat dibuat. Periksa data yang dipilih."
	if errors.Is(err, repository.ErrConflict) {
		message = "Username sudah digunakan atau siswa sudah memiliki akun."
	}
	h.formError(w, r, actor, role, fullName, studentID, status, message)
}

func (h *UserPageHandler) renderPasswordPage(w http.ResponseWriter, r *http.Request, actor service.Actor, message string, updated bool) {
	description := "Perbarui kata sandi akun Anda."
	if actor.PasswordChangeRequired {
		description = "Akun ini menggunakan password sementara. Ganti password untuk melanjutkan menggunakan aplikasi."
	}
	vm := UserPageViewModel{Title: "Ganti Password", Heading: "Ganti Password", Description: description, Role: string(actor.Role), Username: actor.Username, ActiveNav: "change-password", Error: message, PasswordChangeRequired: actor.PasswordChangeRequired}
	if token, ok := middleware.CSRFToken(r); ok {
		vm.CSRFToken = token
	} else {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if updated {
		vm.Notice = "Kata sandi berhasil diperbarui."
	}
	if err := h.templates.ExecuteTemplate(w, "change-password", vm); err != nil {
		log.Printf("render change password page for user %s: %v", actor.UserID, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func normalizeManagedRole(raw string) string { return strings.ToUpper(strings.TrimSpace(raw)) }

func usernameFromName(name string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, strings.TrimSpace(name))
}

func urlQueryEscape(value string) string { return url.QueryEscape(value) }

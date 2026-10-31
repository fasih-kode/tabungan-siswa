package http

import (
	"errors"
	"html/template"
	"mime"
	"net/http"
	"net/url"
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

const maxSavingsFormBytes = 16 << 10

type SavingsDirectoryRow struct {
	StudentID, Name, NIS, Class, StudentStatus string
	AccountID, AccountStatus, Balance          string
	LeavingSettlementURL                       string
	CanCreate                                  bool
}

type SavingsPassbookRow struct {
	ID, Date, Type, Amount, Balance string
}

type SavingsYearOption struct{ ID, Name string }
type SavingsClassOption struct{ ID, Name string }

type SavingsPageViewModel struct {
	Title, Heading, Description, Role, Username, CSRFToken, ActiveNav       string
	Error, Notice, AcademicYearID, ClassID, Search                          string
	AccountOnly                                                             bool
	AccountID, StudentID, StudentName, NIS, AccountStatus, Balance          string
	YearName, SettlementURL, LeavingSettlementURL                           string
	LateRequestURL                                                          string
	LeavingDate                                                             string
	LastLateDate                                                            string
	Page, Limit, PreviousPage, NextPage                                     int
	HasPrevious, HasNext, IsDetail, IsStudent                               bool
	CanTransact, CanSettleLeaving                                           bool
	CanRequestLateTransaction                                               bool
	LeavingSettlementBlocked, Graduated                                     bool
	TransactionURL, TransactionDate, MinTransactionDate, MaxTransactionDate string
	Years                                                                   []SavingsYearOption
	Classes                                                                 []SavingsClassOption
	Rows                                                                    []SavingsDirectoryRow
	Passbook                                                                []SavingsPassbookRow
	Pagination                                                              PaginationViewModel
}

type SavingsPageHandler struct {
	accounts  service.SavingsAccountService
	students  service.StudentService
	years     service.AcademicYearService
	classes   service.ClassService
	templates *template.Template
}

func NewSavingsPageHandler(accounts service.SavingsAccountService, students service.StudentService, years service.AcademicYearService, classes service.ClassService, templates *template.Template) (*SavingsPageHandler, error) {
	if accounts == nil || students == nil || years == nil || classes == nil || templates == nil {
		return nil, service.ErrInvalidDependency
	}
	return &SavingsPageHandler{accounts: accounts, students: students, years: years, classes: classes, templates: templates}, nil
}

func (h *SavingsPageHandler) List(w http.ResponseWriter, r *http.Request) {
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
	vm := h.baseVM(actor)
	csrfToken, ok := middleware.CSRFToken(r)
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	vm.CSRFToken = csrfToken
	yearsOut, err := h.years.List(r.Context(), service.ListAcademicYearsInput{Actor: actor, Options: service.ListOptions{Limit: 250}})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	openYearID := uuid.Nil
	for _, year := range yearsOut.AcademicYears {
		if year == nil {
			continue
		}
		vm.Years = append(vm.Years, SavingsYearOption{ID: year.ID.String(), Name: year.Name})
		if year.IsOpen() {
			openYearID = year.ID
		}
	}
	yearID := strings.TrimSpace(r.URL.Query().Get("academic_year_id"))
	if yearID == "" && openYearID != uuid.Nil {
		yearID = openYearID.String()
	}
	parsedYearID := parseOptionalUUID(yearID)
	if yearID != "" && parsedYearID == uuid.Nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	vm.AcademicYearID = yearID
	if actor.Role == domain.RoleAdmin || actor.Role == domain.RoleWaliKelas {
		vm.SettlementURL = "/savings?view=settlement"
		if parsedYearID != uuid.Nil {
			vm.SettlementURL += "&academic_year_id=" + parsedYearID.String()
		}
	}
	vm.Page, vm.Limit = page, limit
	switch r.URL.Query().Get("notice") {
	case "created":
		vm.Notice = "Tabungan berhasil dibuka."
	case "deposit_created":
		vm.Notice = "Setoran berhasil dicatat."
	case "withdrawal_created":
		vm.Notice = "Penarikan berhasil dicatat."
	}
	for _, year := range yearsOut.AcademicYears {
		if year != nil && year.ID == parsedYearID {
			vm.YearName = year.Name
		}
	}
	if parsedYearID == uuid.Nil {
		vm.Notice = "Belum ada tahun ajaran terbuka. Pilih tahun ajaran untuk melihat riwayat tabungan."
		h.render(w, vm)
		return
	}
	selectedYear, err := h.years.Get(r.Context(), service.GetAcademicYearInput{Actor: actor, AcademicYearID: parsedYearID})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	if selectedYear.AcademicYear != nil {
		vm.YearName = selectedYear.AcademicYear.Name
	}
	if actor.Role == domain.RoleSiswa {
		student, err := h.students.GetByUserID(r.Context(), service.GetStudentByUserIDInput{Actor: actor, UserID: actor.UserID})
		if err != nil {
			httperror.Write(w, err)
			return
		}
		vm.IsStudent = true
		if err := h.appendSavingsRow(r, actor, parsedYearID, student.Student, service.StudentPlacement{}, &vm); err != nil {
			httperror.Write(w, err)
			return
		}
		h.render(w, vm)
		return
	}
	if actor.Role != domain.RoleAdmin && actor.Role != domain.RoleWaliKelas {
		httperror.Write(w, service.ErrForbidden)
		return
	}
	classID := strings.TrimSpace(r.URL.Query().Get("class_id"))
	parsedClassID := parseOptionalUUID(classID)
	if classID != "" && (parsedClassID == uuid.Nil || parsedYearID == uuid.Nil || (actor.Role != domain.RoleWaliKelas && actor.Role != domain.RoleAdmin)) {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	vm.ClassID, vm.Search = classID, strings.TrimSpace(r.URL.Query().Get("q"))
	vm.AccountOnly = actor.Role == domain.RoleAdmin && r.URL.Query().Get("account_only") == "1"
	includePlacement := actor.Role == domain.RoleWaliKelas || actor.Role == domain.RoleAdmin
	if parsedYearID != uuid.Nil && includePlacement {
		classesOut, err := h.classes.List(r.Context(), service.ListClassesInput{Actor: actor, AcademicYearID: parsedYearID, Options: service.ListOptions{Limit: 250}})
		if err != nil {
			httperror.Write(w, err)
			return
		}
		for _, class := range classesOut.Classes {
			if class != nil {
				vm.Classes = append(vm.Classes, SavingsClassOption{ID: class.ID.String(), Name: class.Name})
			}
		}
	}
	queryStudents := func(offset, batchLimit int) (service.ListStudentsOutput, error) {
		return h.students.List(r.Context(), service.ListStudentsInput{Actor: actor, IncludePlacement: includePlacement, AcademicYearID: parsedYearID, ClassID: parsedClassID, Options: service.ListOptions{Limit: batchLimit, Offset: offset, IncludeTotal: true, Search: vm.Search}})
	}
	studentsOut, err := queryStudents((page-1)*limit, limit+1)
	if err != nil {
		httperror.Write(w, err)
		return
	}
	vm.Rows = make([]SavingsDirectoryRow, 0, len(studentsOut.Students))
	if !vm.AccountOnly {
		for _, student := range studentsOut.Students {
			if student == nil {
				continue
			}
			placement := studentsOut.Placements[student.ID]
			rowVM := SavingsDirectoryRow{StudentID: student.ID.String(), Name: student.Name, NIS: stringValue(student.NIS), StudentStatus: string(student.Status), Class: placement.Class}
			account, accountErr := h.accounts.GetByStudentAndAcademicYear(r.Context(), service.GetSavingsAccountByStudentAndAcademicYearInput{Actor: actor, StudentID: student.ID, AcademicYearID: parsedYearID})
			if accountErr == nil && account.Account != nil {
				rowVM.AccountID, rowVM.AccountStatus = account.Account.ID.String(), interfaceStatusLabel(string(account.Account.Status))
				if actor.Role == domain.RoleWaliKelas && student.IsLeft() && account.Account.IsOpen() {
					if selectedYear.AcademicYear != nil && !selectedYear.AcademicYear.IsClosed() {
						rowVM.LeavingSettlementURL = "/savings/" + account.Account.ID.String() + "/leaving-settlement-preview"
					} else {
						rowVM.AccountStatus = "Penyelesaian tertunda (tahun ditutup)"
					}
				}
				balance, balanceErr := h.accounts.GetBalance(r.Context(), service.GetSavingsAccountBalanceInput{Actor: actor, SavingsAccountID: account.Account.ID})
				if balanceErr != nil {
					httperror.Write(w, balanceErr)
					return
				}
				rowVM.Balance = formatRupiah(balance.Balance)
			} else if errors.Is(accountErr, repository.ErrNotFound) {
				rowVM.CanCreate = actor.Role == domain.RoleWaliKelas && selectedYear.AcademicYear != nil && selectedYear.AcademicYear.IsOpen() && student.IsActive()
			} else if accountErr != nil {
				httperror.Write(w, accountErr)
				return
			}
			vm.Rows = append(vm.Rows, rowVM)
		}
	} else {
		// Filter by account before pagination so every page contains account holders.
		// Walk the scoped roster in bounded batches; account ownership is checked
		// by the account service for the selected academic year.
		vm.Rows = nil
		const batchSize = 100
		for offset := 0; offset < studentsOut.Total; offset += batchSize {
			batch, listErr := queryStudents(offset, batchSize)
			if listErr != nil {
				httperror.Write(w, listErr)
				return
			}
			for _, student := range batch.Students {
				if student == nil {
					continue
				}
				account, accountErr := h.accounts.GetByStudentAndAcademicYear(r.Context(), service.GetSavingsAccountByStudentAndAcademicYearInput{Actor: actor, StudentID: student.ID, AcademicYearID: parsedYearID})
				if errors.Is(accountErr, repository.ErrNotFound) {
					continue
				}
				if accountErr != nil {
					httperror.Write(w, accountErr)
					return
				}
				if account.Account == nil {
					continue
				}
				placement := batch.Placements[student.ID]
				vm.Rows = append(vm.Rows, SavingsDirectoryRow{StudentID: student.ID.String(), Name: student.Name, NIS: stringValue(student.NIS), StudentStatus: string(student.Status), Class: placement.Class, AccountID: account.Account.ID.String(), AccountStatus: interfaceStatusLabel(string(account.Account.Status))})
			}
		}
		studentsOut.Total = len(vm.Rows)
		start := (page - 1) * limit
		if start > len(vm.Rows) {
			start = len(vm.Rows)
		}
		end := start + limit
		if end > len(vm.Rows) {
			end = len(vm.Rows)
		}
		vm.Rows = vm.Rows[start:end]
		for i := range vm.Rows {
			accountID, parseErr := uuid.Parse(vm.Rows[i].AccountID)
			if parseErr != nil {
				http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			balance, balanceErr := h.accounts.GetBalance(r.Context(), service.GetSavingsAccountBalanceInput{Actor: actor, SavingsAccountID: accountID})
			if balanceErr != nil {
				httperror.Write(w, balanceErr)
				return
			}
			vm.Rows[i].Balance = formatRupiah(balance.Balance)
		}
	}
	vm.HasNext = len(vm.Rows) > limit
	if vm.HasNext {
		vm.Rows = vm.Rows[:limit]
	}
	vm.HasPrevious, vm.PreviousPage, vm.NextPage = page > 1, page-1, page+1
	vm.Pagination = newPagination(r, page, studentsOut.Total, len(vm.Rows), vm.HasNext)
	if redirectMissingPage(w, r, vm.Pagination) {
		return
	}
	h.render(w, vm)
}

func (h *SavingsPageHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if err := service.RequireRole(actor, domain.RoleWaliKelas); err != nil {
		httperror.Write(w, err)
		return
	}
	if err := parseSavingsForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	studentID, errStudent := uuid.Parse(strings.TrimSpace(r.PostForm.Get("student_id")))
	yearID, errYear := uuid.Parse(strings.TrimSpace(r.PostForm.Get("academic_year_id")))
	if errStudent != nil || errYear != nil || studentID == uuid.Nil || yearID == uuid.Nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if _, err := h.accounts.Create(r.Context(), service.CreateSavingsAccountInput{Actor: actor, StudentID: studentID, AcademicYearID: yearID}); err != nil {
		httperror.Write(w, err)
		return
	}
	http.Redirect(w, r, "/savings?academic_year_id="+url.QueryEscape(yearID.String())+"&notice=created", http.StatusSeeOther)
}

// Parse before CSRF middleware can cache form data or discard parsing errors.
func validateSavingsRequestForm(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := parseSavingsForm(w, r); err != nil {
			http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func parseSavingsForm(w http.ResponseWriter, r *http.Request) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		return errors.New("invalid savings form content type")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSavingsFormBytes)
	return r.ParseForm()
}

func (h *SavingsPageHandler) Detail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	accountID, err := uuid.Parse(r.PathValue("id"))
	if err != nil || accountID == uuid.Nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	account, err := h.accounts.Get(r.Context(), service.GetSavingsAccountInput{Actor: actor, SavingsAccountID: accountID})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	if account.Account == nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	studentOut, err := h.students.Get(r.Context(), service.GetStudentInput{Actor: actor, StudentID: account.Account.StudentID, AcademicYearID: account.Account.AcademicYearID})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	if studentOut.Student == nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	passbook, err := h.accounts.GetPassbook(r.Context(), service.GetSavingsAccountPassbookInput{Actor: actor, SavingsAccountID: accountID})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	yearOut, err := h.years.Get(r.Context(), service.GetAcademicYearInput{Actor: actor, AcademicYearID: account.Account.AcademicYearID})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	if yearOut.AcademicYear == nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	vm := h.baseVM(actor)
	csrfToken, ok := middleware.CSRFToken(r)
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	vm.CSRFToken = csrfToken
	vm.IsDetail, vm.AccountID, vm.StudentID = true, accountID.String(), account.Account.StudentID.String()
	vm.StudentName, vm.NIS, vm.AccountStatus = studentOut.Student.Name, stringValue(studentOut.Student.NIS), interfaceStatusLabel(string(account.Account.Status))
	vm.AcademicYearID, vm.YearName = account.Account.AcademicYearID.String(), yearOut.AcademicYear.Name
	vm.TransactionURL = "/savings/" + accountID.String() + "/transactions"
	vm.MinTransactionDate = yearOut.AcademicYear.StartDate.Format("2006-01-02")
	vm.MaxTransactionDate = yearOut.AcademicYear.EndDate.AddDate(0, 0, -1).Format("2006-01-02")
	today := time.Now().UTC()
	if today.Before(yearOut.AcademicYear.StartDate) || !today.Before(yearOut.AcademicYear.EndDate) {
		today = yearOut.AcademicYear.StartDate
	}
	vm.TransactionDate = today.Format("2006-01-02")
	vm.CanTransact = actor.Role == domain.RoleWaliKelas &&
		studentOut.Student.IsActive() && account.Account.IsOpen() && yearOut.AcademicYear.IsOpen()
	vm.Graduated = actor.Role == domain.RoleWaliKelas && studentOut.Student.IsGraduated() && account.Account.IsOpen()
	vm.LeavingSettlementBlocked = actor.Role == domain.RoleWaliKelas && studentOut.Student.IsLeft() && account.Account.IsOpen() && yearOut.AcademicYear.IsClosed()
	vm.CanSettleLeaving = actor.Role == domain.RoleWaliKelas && studentOut.Student.IsLeft() && account.Account.IsOpen() && !yearOut.AcademicYear.IsClosed()
	vm.CanRequestLateTransaction = actor.Role == domain.RoleWaliKelas && studentOut.Student.IsLeft() && studentOut.Student.LeavingDate != nil && account.Account.IsOpen() && !yearOut.AcademicYear.IsClosed()
	if studentOut.Student.LeavingDate != nil {
		vm.LeavingDate = studentOut.Student.LeavingDate.Format("2006-01-02")
		vm.LastLateDate = studentOut.Student.LeavingDate.AddDate(0, 0, -1).Format("2006-01-02")
	}
	if vm.CanRequestLateTransaction {
		vm.LateRequestURL = "/savings/" + accountID.String() + "/late-transaction-requests"
	}
	if vm.CanSettleLeaving {
		vm.LeavingSettlementURL = "/savings/" + accountID.String() + "/leaving-settlement-preview"
	}
	vm.Balance = formatRupiah(passbook.Balance)
	for _, entry := range passbook.Entries {
		if entry == nil || entry.Transaction == nil {
			continue
		}
		tx := entry.Transaction
		label := "Setor"
		if tx.Type == domain.TransactionWithdrawal {
			label = "Tarik"
		}
		vm.Passbook = append(vm.Passbook, SavingsPassbookRow{ID: tx.ID.String(), Date: tx.TransactionDate.Format("2006-01-02"), Type: label, Amount: formatRupiah(tx.Amount), Balance: formatRupiah(entry.RunningBalance)})
	}
	h.render(w, vm)
}

func (h *SavingsPageHandler) baseVM(actor service.Actor) SavingsPageViewModel {
	heading, description := "Tabungan Siswa", "Siklus tabungan, saldo, dan buku tabungan per tahun ajaran."
	if actor.Role == domain.RoleAdmin {
		heading, description = "Histori Tabungan", "Cari buku tabungan siswa berdasarkan tahun ajaran dan kelas."
	}
	return SavingsPageViewModel{Title: heading, Heading: heading, Description: description, Role: string(actor.Role), Username: actor.Username, ActiveNav: "savings"}
}

func (h *SavingsPageHandler) render(w http.ResponseWriter, vm SavingsPageViewModel) {
	if err := h.templates.ExecuteTemplate(w, "savings", vm); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *SavingsPageHandler) appendSavingsRow(r *http.Request, actor service.Actor, yearID uuid.UUID, student *domain.Student, placement service.StudentPlacement, vm *SavingsPageViewModel) error {
	if student == nil {
		return service.ErrInvalidDependency
	}
	row := SavingsDirectoryRow{StudentID: student.ID.String(), Name: student.Name, NIS: stringValue(student.NIS), StudentStatus: string(student.Status), Class: placement.Class}
	account, err := h.accounts.GetByStudentAndAcademicYear(r.Context(), service.GetSavingsAccountByStudentAndAcademicYearInput{Actor: actor, StudentID: student.ID, AcademicYearID: yearID})
	if errors.Is(err, repository.ErrNotFound) {
		vm.Rows = append(vm.Rows, row)
		return nil
	}
	if err != nil {
		return err
	}
	if account.Account != nil {
		row.AccountID, row.AccountStatus = account.Account.ID.String(), interfaceStatusLabel(string(account.Account.Status))
		balance, err := h.accounts.GetBalance(r.Context(), service.GetSavingsAccountBalanceInput{Actor: actor, SavingsAccountID: account.Account.ID})
		if err != nil {
			return err
		}
		row.Balance = formatRupiah(balance.Balance)
	}
	vm.Rows = append(vm.Rows, row)
	return nil
}

func formatRupiah(amount domain.Money) string {
	value := strconv.FormatInt(int64(amount), 10)
	if len(value) <= 3 {
		return "Rp. " + value
	}
	var groups []string
	for len(value) > 3 {
		groups = append([]string{value[len(value)-3:]}, groups...)
		value = value[:len(value)-3]
	}
	groups = append([]string{value}, groups...)
	return "Rp. " + strings.Join(groups, ".")
}

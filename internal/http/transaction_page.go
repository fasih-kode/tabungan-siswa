package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"mime"
	"net/http"
	"net/url"
	"sort"
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

const maxTransactionFormBytes = 16 << 10

type TransactionPageRow struct {
	ID, Date, Type, Amount, Balance, Status string
}

type TransactionAuditView struct {
	Actor, Action, At, Reason, Changes string
	AtTime                             time.Time
}

type TransactionPageViewModel struct {
	Title, Heading, Description, Role, Username, CSRFToken, ActiveNav string
	Error, Notice, AccountID, TransactionID, StudentName, NIS         string
	YearName, AccountStatus, AccountBalance                           string
	TransactionDate, TransactionType, Amount, RecordedAt              string
	AmountValue, TransactionStatus                                    string
	Reason                                                            string
	Page, Limit                                                       int
	IsDetail, CanWrite                                                bool
	CanAudit                                                          bool
	Pagination                                                        PaginationViewModel
	Rows                                                              []TransactionPageRow
	Audit                                                             []TransactionAuditView
}

type TransactionPageHandler struct {
	transactions service.TransactionService
	accounts     service.SavingsAccountService
	students     service.StudentService
	years        service.AcademicYearService
	audit        service.AuditService
	templates    *template.Template
}

func NewTransactionPageHandler(
	transactions service.TransactionService,
	accounts service.SavingsAccountService,
	students service.StudentService,
	years service.AcademicYearService,
	templates *template.Template,
	auditServices ...service.AuditService,
) (*TransactionPageHandler, error) {
	if transactions == nil || accounts == nil || students == nil || years == nil || templates == nil {
		return nil, service.ErrInvalidDependency
	}
	return &TransactionPageHandler{
		transactions: transactions,
		accounts:     accounts,
		students:     students,
		years:        years,
		audit:        firstAuditService(auditServices),
		templates:    templates,
	}, nil
}

func (h *TransactionPageHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	accountID, err := parseTransactionPathUUID(r, "account_id")
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	page, err := parseTransactionPage(r)
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	account, student, year, err := h.loadAccountContext(r, actor, accountID)
	if err != nil {
		httperror.Write(w, err)
		return
	}
	out, err := h.transactions.List(r.Context(), service.ListTransactionsInput{
		Actor: actor, SavingsAccountID: accountID,
		Options: service.ListOptions{Limit: tablePageSize, Offset: (page - 1) * tablePageSize},
	})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	csrf, ok := middleware.CSRFToken(r)
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	vm := h.baseVM(actor)
	vm.AccountID, vm.StudentName, vm.NIS = accountID.String(), student.Name, stringValue(student.NIS)
	vm.YearName, vm.AccountStatus, vm.CSRFToken = year.Name, interfaceStatusLabel(string(account.Status)), csrf
	vm.AccountBalance = formatRupiah(out.Balance)
	vm.Page, vm.Limit = page, tablePageSize
	vm.Rows = make([]TransactionPageRow, 0, len(out.Transactions))
	for _, tx := range out.Transactions {
		if tx == nil {
			continue
		}
		vm.Rows = append(vm.Rows, TransactionPageRow{
			ID: tx.ID.String(), Date: tx.TransactionDate.Format("2006-01-02"),
			Type: transactionTypeLabel(tx.Type), Amount: formatRupiah(tx.Amount),
			Balance: formatTransactionBalance(out.RunningBalances[tx.ID], tx.Status), Status: interfaceStatusLabel(string(tx.Status)),
		})
	}
	hasNext := page*tablePageSize < out.Total
	vm.Pagination = newPagination(r, page, out.Total, len(vm.Rows), hasNext)
	if redirectMissingPage(w, r, vm.Pagination) {
		return
	}
	h.render(w, vm)
}

func (h *TransactionPageHandler) Detail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, "GET, HEAD")
		return
	}
	actor, ok := service.ActorFromContext(r.Context())
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	transactionID, err := parseTransactionPathUUID(r, "transaction_id")
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	out, err := h.transactions.Get(r.Context(), service.GetTransactionInput{Actor: actor, TransactionID: transactionID})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	if out.Transaction == nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	tx := out.Transaction
	account, student, year, err := h.loadAccountContext(r, actor, tx.SavingsAccountID)
	if err != nil {
		httperror.Write(w, err)
		return
	}
	balance, err := h.accounts.GetBalance(r.Context(), service.GetSavingsAccountBalanceInput{Actor: actor, SavingsAccountID: account.ID})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	csrf, ok := middleware.CSRFToken(r)
	if !ok {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	vm := h.baseVM(actor)
	vm.IsDetail, vm.TransactionID, vm.AccountID = true, tx.ID.String(), account.ID.String()
	vm.StudentName, vm.NIS, vm.YearName = student.Name, stringValue(student.NIS), year.Name
	vm.AccountStatus, vm.AccountBalance, vm.CSRFToken = interfaceStatusLabel(string(account.Status)), formatRupiah(balance.Balance), csrf
	vm.TransactionDate, vm.TransactionType = tx.TransactionDate.Format("2006-01-02"), transactionTypeLabel(tx.Type)
	vm.TransactionStatus, vm.AmountValue = interfaceStatusLabel(string(tx.Status)), strconv.FormatInt(int64(tx.Amount), 10)
	vm.Amount, vm.RecordedAt = formatRupiah(tx.Amount), tx.CreatedAt.Format(time.RFC3339)
	vm.CanAudit = h.audit != nil && (actor.Role == domain.RoleAdmin || actor.Role == domain.RoleWaliKelas)
	if vm.CanAudit {
		logs, auditErr := h.audit.ListByEntity(r.Context(), service.ListAuditByEntityInput{
			Actor: actor, EntityType: domain.AuditEntityTransaction, EntityID: tx.ID,
			Options: service.ListOptions{Limit: 100},
		})
		if auditErr != nil {
			httperror.Write(w, auditErr)
			return
		}
		vm.Audit = make([]TransactionAuditView, 0, len(logs.AuditLogs))
		for i := len(logs.AuditLogs) - 1; i >= 0; i-- {
			entry := logs.AuditLogs[i]
			if entry != nil {
				vm.Audit = append(vm.Audit, transactionAuditView(*entry))
			}
		}
		accountLogs, accountAuditErr := h.audit.ListByEntity(r.Context(), service.ListAuditByEntityInput{
			Actor: actor, EntityType: domain.AuditEntitySavingsAccount, EntityID: account.ID,
			Options: service.ListOptions{Limit: 100},
		})
		if accountAuditErr != nil {
			httperror.Write(w, accountAuditErr)
			return
		}
		for i := len(accountLogs.AuditLogs) - 1; i >= 0; i-- {
			if entry := accountLogs.AuditLogs[i]; entry != nil {
				vm.Audit = append(vm.Audit, transactionAuditView(*entry))
			}
		}
		sort.SliceStable(vm.Audit, func(i, j int) bool { return vm.Audit[i].AtTime.Before(vm.Audit[j].AtTime) })
	}
	vm.CanWrite = actor.Role == domain.RoleWaliKelas && tx.Status == domain.TransactionActive && account.IsOpen() && year.IsOpen()
	if !vm.CanWrite && actor.Role == domain.RoleWaliKelas && tx.Status == domain.TransactionActive && account.IsOpen() && year.IsClosing() {
		if checker, ok := h.transactions.(interface {
			CanCorrect(context.Context, service.Actor, uuid.UUID) (bool, error)
		}); ok {
			allowed, checkErr := checker.CanCorrect(r.Context(), actor, tx.ID)
			if checkErr != nil {
				httperror.Write(w, checkErr)
				return
			}
			vm.CanWrite = allowed
		}
	}
	switch r.URL.Query().Get("notice") {
	case "transaction_edited":
		vm.Notice = "Transaksi berhasil dikoreksi."
	case "transaction_cancelled":
		vm.Notice = "Transaksi berhasil dibatalkan."
	}
	h.render(w, vm)
}

func (h *TransactionPageHandler) Edit(w http.ResponseWriter, r *http.Request) { h.correct(w, r, false) }
func (h *TransactionPageHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	h.correct(w, r, true)
}

func (h *TransactionPageHandler) correct(w http.ResponseWriter, r *http.Request, cancel bool) {
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
	if err := parseCorrectionTransactionForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	id, err := parseTransactionPathUUID(r, "transaction_id")
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	reason := r.PostForm.Get("reason")
	if cancel {
		_, err = h.transactions.Cancel(r.Context(), service.CancelTransactionInput{Actor: actor, TransactionID: id, Reason: reason})
	} else {
		amount, amountErr := strconv.ParseInt(r.PostForm.Get("amount"), 10, 64)
		date, dateErr := time.Parse("2006-01-02", r.PostForm.Get("transaction_date"))
		if amountErr != nil || amount <= 0 || amount > int64(domain.MaxMoney) || dateErr != nil {
			http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		_, err = h.transactions.Edit(r.Context(), service.EditTransactionInput{Actor: actor, TransactionID: id, Amount: domain.Money(amount), TransactionDate: date, Reason: reason})
	}
	if err != nil {
		writeCorrectionError(w, err)
		return
	}
	notice := "transaction_edited"
	if cancel {
		notice = "transaction_cancelled"
	}
	http.Redirect(w, r, "/transactions/"+id.String()+"?notice="+url.QueryEscape(notice), http.StatusSeeOther)
}

func writeCorrectionError(w http.ResponseWriter, err error) {
	status := httperror.Status(err)
	message := localizedStatusMessage(status)
	switch {
	case errors.Is(err, service.ErrTransactionDateOutOfAcademicYear):
		message = "Tanggal transaksi harus berada dalam periode tahun ajaran."
	case errors.Is(err, domain.ErrInvalidTransition):
		message = "Koreksi hanya tersedia untuk rekening dan tahun yang terbuka atau melalui jendela koreksi yang disetujui."
	case errors.Is(err, domain.ErrNegativeBalance):
		message = "Koreksi ditolak karena saldo transaksi berikutnya akan menjadi negatif."
	case errors.Is(err, domain.ErrAlreadyCancelled):
		message = "Transaksi ini sudah dibatalkan."
	case errors.Is(err, repository.ErrNotFound):
		message = localizedStatusMessage(http.StatusNotFound)
	}
	http.Error(w, message, status)
}

func firstAuditService(services []service.AuditService) service.AuditService {
	if len(services) > 0 {
		return services[0]
	}
	return nil
}
func formatTransactionBalance(balance domain.Money, status domain.TransactionStatus) string {
	if status == domain.TransactionCancelled {
		return "—"
	}
	return formatRupiah(balance)
}
func transactionAuditView(log domain.AuditLog) TransactionAuditView {
	view := TransactionAuditView{Action: log.Action, At: log.OccurredAt.Format(time.RFC3339), AtTime: log.OccurredAt}
	if log.ActorUserID != nil {
		view.Actor = log.ActorUserID.String()
	} else {
		view.Actor = "Sistem"
	}
	var after struct {
		Reason      string             `json:"reason"`
		Transaction domain.Transaction `json:"transaction"`
	}
	_ = json.Unmarshal(log.AfterData, &after)
	view.Reason = after.Reason
	if len(log.BeforeData) > 0 && after.Transaction.ID != uuid.Nil {
		var before domain.Transaction
		if json.Unmarshal(log.BeforeData, &before) == nil {
			view.Changes = fmt.Sprintf("Jumlah: %s → %s; tanggal: %s → %s", formatRupiah(before.Amount), formatRupiah(after.Transaction.Amount), before.TransactionDate.Format("2006-01-02"), after.Transaction.TransactionDate.Format("2006-01-02"))
		}
	}
	return view
}

func (h *TransactionPageHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	h.record(w, r, true)
}

func (h *TransactionPageHandler) Withdrawal(w http.ResponseWriter, r *http.Request) {
	h.record(w, r, false)
}

func (h *TransactionPageHandler) record(w http.ResponseWriter, r *http.Request, deposit bool) {
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
	if err := parseTransactionForm(w, r); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	accountID, err := parseTransactionPathUUID(r, "account_id")
	if err != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	amount, errAmount := strconv.ParseInt(r.PostForm.Get("amount"), 10, 64)
	date, errDate := time.Parse("2006-01-02", r.PostForm.Get("transaction_date"))
	if errAmount != nil || amount <= 0 || amount > int64(domain.MaxMoney) || errDate != nil {
		http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if deposit {
		_, err = h.transactions.Deposit(r.Context(), service.DepositInput{
			Actor: actor, SavingsAccountID: accountID, Amount: domain.Money(amount), TransactionDate: date,
		})
	} else {
		_, err = h.transactions.Withdrawal(r.Context(), service.WithdrawalInput{
			Actor: actor, SavingsAccountID: accountID, Amount: domain.Money(amount), TransactionDate: date,
		})
	}
	if err != nil {
		httperror.Write(w, err)
		return
	}
	notice := "deposit_created"
	if !deposit {
		notice = "withdrawal_created"
	}
	http.Redirect(w, r, "/savings/"+accountID.String()+"?notice="+url.QueryEscape(notice), http.StatusSeeOther)
}

func (h *TransactionPageHandler) loadAccountContext(
	r *http.Request,
	actor service.Actor,
	accountID uuid.UUID,
) (domain.SavingsAccount, domain.Student, domain.AcademicYear, error) {
	accountOut, err := h.accounts.Get(r.Context(), service.GetSavingsAccountInput{Actor: actor, SavingsAccountID: accountID})
	if err != nil {
		return domain.SavingsAccount{}, domain.Student{}, domain.AcademicYear{}, err
	}
	if accountOut.Account == nil {
		return domain.SavingsAccount{}, domain.Student{}, domain.AcademicYear{}, service.ErrInvalidDependency
	}
	studentOut, err := h.students.Get(r.Context(), service.GetStudentInput{
		Actor: actor, StudentID: accountOut.Account.StudentID, AcademicYearID: accountOut.Account.AcademicYearID,
	})
	if err != nil {
		return domain.SavingsAccount{}, domain.Student{}, domain.AcademicYear{}, err
	}
	yearOut, err := h.years.Get(r.Context(), service.GetAcademicYearInput{Actor: actor, AcademicYearID: accountOut.Account.AcademicYearID})
	if err != nil {
		return domain.SavingsAccount{}, domain.Student{}, domain.AcademicYear{}, err
	}
	if studentOut.Student == nil || yearOut.AcademicYear == nil {
		return domain.SavingsAccount{}, domain.Student{}, domain.AcademicYear{}, service.ErrInvalidDependency
	}
	return *accountOut.Account, *studentOut.Student, *yearOut.AcademicYear, nil
}

func (h *TransactionPageHandler) baseVM(actor service.Actor) TransactionPageViewModel {
	return TransactionPageViewModel{
		Title: "Transaksi", Heading: "Transaksi Tabungan",
		Description: "Riwayat transaksi dan saldo berjalan untuk siklus tabungan.",
		Role:        string(actor.Role), Username: actor.Username, ActiveNav: "savings",
	}
}

func (h *TransactionPageHandler) render(w http.ResponseWriter, vm TransactionPageViewModel) {
	if err := h.templates.ExecuteTemplate(w, "transactions", vm); err != nil {
		http.Error(w, localizedStatusMessage(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func parseTransactionPathUUID(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(r.PathValue(name)))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, errors.New("invalid resource ID")
	}
	return id, nil
}

func parseTransactionPage(r *http.Request) (int, error) {
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return 0, errors.New("invalid page")
		}
		page = value
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			return 0, errors.New("invalid limit")
		}
	}
	maxInt := int(^uint(0) >> 1)
	if page-1 > (maxInt-tablePageSize)/tablePageSize {
		return 0, errors.New("page offset overflow")
	}
	return page, nil
}

func validateTransactionRequestForm(next http.Handler) http.Handler {
	return validateTransactionForm(next, false)
}

func validateTransactionCorrectionForm(next http.Handler) http.Handler {
	return validateTransactionForm(next, true)
}

func validateTransactionForm(next http.Handler, allowReason bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := service.ActorFromContext(r.Context())
		if !ok {
			http.Error(w, localizedStatusMessage(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		if err := service.RequireRole(actor, domain.RoleWaliKelas); err != nil {
			httperror.Write(w, err)
			return
		}
		if err := parseTransactionFormFields(w, r, allowReason); err != nil {
			http.Error(w, localizedStatusMessage(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func parseTransactionForm(w http.ResponseWriter, r *http.Request) error {
	return parseTransactionFormFields(w, r, false)
}

func parseCorrectionTransactionForm(w http.ResponseWriter, r *http.Request) error {
	return parseTransactionFormFields(w, r, true)
}

func parseTransactionFormFields(w http.ResponseWriter, r *http.Request, allowReason bool) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		return errors.New("invalid transaction form content type")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxTransactionFormBytes)
	if err := r.ParseForm(); err != nil {
		return err
	}
	for key, values := range r.PostForm {
		if key != "amount" && key != "transaction_date" && key != "csrf_token" && !(allowReason && key == "reason") {
			return errors.New("unsupported transaction form field")
		}
		if len(values) != 1 {
			return errors.New("repeated transaction form field")
		}
	}
	return nil
}

func transactionTypeLabel(kind domain.TransactionType) string {
	if kind == domain.TransactionWithdrawal {
		return "Penarikan"
	}
	return "Setoran"
}

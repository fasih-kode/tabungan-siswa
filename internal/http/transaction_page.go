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
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

const maxTransactionFormBytes = 16 << 10

type TransactionPageRow struct {
	ID, Date, Type, Amount, Balance string
}

type TransactionPageViewModel struct {
	Title, Heading, Description, Role, Username, CSRFToken, ActiveNav string
	Error, Notice, AccountID, TransactionID, StudentName, NIS         string
	YearName, AccountStatus, AccountBalance                           string
	TransactionDate, TransactionType, Amount, RecordedAt              string
	Page, Limit                                                       int
	IsDetail, CanWrite                                                bool
	Pagination                                                        PaginationViewModel
	Rows                                                              []TransactionPageRow
}

type TransactionPageHandler struct {
	transactions service.TransactionService
	accounts     service.SavingsAccountService
	students     service.StudentService
	years        service.AcademicYearService
	templates    *template.Template
}

func NewTransactionPageHandler(
	transactions service.TransactionService,
	accounts service.SavingsAccountService,
	students service.StudentService,
	years service.AcademicYearService,
	templates *template.Template,
) (*TransactionPageHandler, error) {
	if transactions == nil || accounts == nil || students == nil || years == nil || templates == nil {
		return nil, service.ErrInvalidDependency
	}
	return &TransactionPageHandler{
		transactions: transactions,
		accounts:     accounts,
		students:     students,
		years:        years,
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
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	accountID, err := parseTransactionPathUUID(r, "account_id")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	page, err := parseTransactionPage(r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
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
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	vm := h.baseVM(actor)
	vm.AccountID, vm.StudentName, vm.NIS = accountID.String(), student.Name, stringValue(student.NIS)
	vm.YearName, vm.AccountStatus, vm.CSRFToken = year.Name, string(account.Status), csrf
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
			Balance: formatRupiah(out.RunningBalances[tx.ID]),
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
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	transactionID, err := parseTransactionPathUUID(r, "transaction_id")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	out, err := h.transactions.Get(r.Context(), service.GetTransactionInput{Actor: actor, TransactionID: transactionID})
	if err != nil {
		httperror.Write(w, err)
		return
	}
	if out.Transaction == nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	vm := h.baseVM(actor)
	vm.IsDetail, vm.TransactionID, vm.AccountID = true, tx.ID.String(), account.ID.String()
	vm.StudentName, vm.NIS, vm.YearName = student.Name, stringValue(student.NIS), year.Name
	vm.AccountStatus, vm.AccountBalance, vm.CSRFToken = string(account.Status), formatRupiah(balance.Balance), csrf
	vm.TransactionDate, vm.TransactionType = tx.TransactionDate.Format("2006-01-02"), transactionTypeLabel(tx.Type)
	vm.Amount, vm.RecordedAt = formatRupiah(tx.Amount), tx.CreatedAt.Format(time.RFC3339)
	h.render(w, vm)
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
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if err := service.RequireRole(actor, domain.RoleAdmin, domain.RoleWaliKelas); err != nil {
		httperror.Write(w, err)
		return
	}
	if err := parseTransactionForm(w, r); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	accountID, err := parseTransactionPathUUID(r, "account_id")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	amount, errAmount := strconv.ParseInt(r.PostForm.Get("amount"), 10, 64)
	date, errDate := time.Parse("2006-01-02", r.PostForm.Get("transaction_date"))
	if errAmount != nil || amount <= 0 || amount > int64(domain.MaxMoney) || errDate != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
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
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := service.ActorFromContext(r.Context())
		if !ok {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		if err := service.RequireRole(actor, domain.RoleAdmin, domain.RoleWaliKelas); err != nil {
			httperror.Write(w, err)
			return
		}
		if err := parseTransactionForm(w, r); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func parseTransactionForm(w http.ResponseWriter, r *http.Request) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		return errors.New("invalid transaction form content type")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxTransactionFormBytes)
	if err := r.ParseForm(); err != nil {
		return err
	}
	for key, values := range r.PostForm {
		if key != "amount" && key != "transaction_date" && key != "csrf_token" {
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

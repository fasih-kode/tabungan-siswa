package http

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/http/middleware"
	"github.com/fasih/tabungan-siswa/internal/security"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

type transactionPageServiceFake struct {
	service.TransactionService
	listOutput                    service.ListTransactionsOutput
	getOutput                     service.GetTransactionOutput
	listInput                     service.ListTransactionsInput
	getInput                      service.GetTransactionInput
	depositInput                  service.DepositInput
	withdrawalInput               service.WithdrawalInput
	editInput                     service.EditTransactionInput
	cancelInput                   service.CancelTransactionInput
	depositErr, withdrawalErr     error
	depositCalls, withdrawalCalls int
	editCalls, cancelCalls        int
}

func (f *transactionPageServiceFake) List(_ context.Context, in service.ListTransactionsInput) (service.ListTransactionsOutput, error) {
	f.listInput = in
	return f.listOutput, nil
}
func (f *transactionPageServiceFake) Get(_ context.Context, in service.GetTransactionInput) (service.GetTransactionOutput, error) {
	f.getInput = in
	return f.getOutput, nil
}
func (f *transactionPageServiceFake) Deposit(_ context.Context, in service.DepositInput) (service.DepositOutput, error) {
	f.depositCalls++
	f.depositInput = in
	return service.DepositOutput{}, f.depositErr
}
func (f *transactionPageServiceFake) Withdrawal(_ context.Context, in service.WithdrawalInput) (service.WithdrawalOutput, error) {
	f.withdrawalCalls++
	f.withdrawalInput = in
	return service.WithdrawalOutput{}, f.withdrawalErr
}
func (f *transactionPageServiceFake) Edit(_ context.Context, in service.EditTransactionInput) (service.EditTransactionOutput, error) {
	f.editCalls++
	f.editInput = in
	return service.EditTransactionOutput{}, nil
}
func (f *transactionPageServiceFake) Cancel(_ context.Context, in service.CancelTransactionInput) (service.CancelTransactionOutput, error) {
	f.cancelCalls++
	f.cancelInput = in
	return service.CancelTransactionOutput{}, nil
}

type transactionPageAccountsFake struct {
	service.SavingsAccountService
	account  domain.SavingsAccount
	getInput service.GetSavingsAccountInput
}

func TestTransactionCorrectionRoutesRequireCSRFAndAcceptAllowedFields(t *testing.T) {
	account := domain.SavingsAccount{ID: uuid.New(), StudentID: uuid.New(), AcademicYearID: uuid.New(), Status: domain.SavingsAccountOpen}
	tx := &transactionPageServiceFake{}
	handler := newTransactionPageHandlerForTest(t, tx, account)
	userID := uuid.New()
	sessionToken := "correction-session-" + uuid.NewString()
	session := domain.Session{ID: uuid.New(), UserID: userID, TokenHash: security.HashSessionToken(sessionToken), ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now()}
	user := domain.User{ID: userID, Username: "wali-correction", Role: domain.RoleWaliKelas}
	auth, err := middleware.NewAuthenticationMiddleware(security.NewDefaultSessionCookie(), &routerSessionRepositoryFake{session: session}, &routerUserRepositoryFake{user: user})
	if err != nil {
		t.Fatal(err)
	}
	templates := template.Must(template.New("pages").Parse(`{{define "transactions"}}{{.StudentName}}{{end}}`))
	dashboard, _ := NewDashboardPageHandler(templates)
	router, err := NewRouter(HandlerSet{Authentication: &AuthenticationHandler{}, LoginPage: newTestLoginPageHandler(t), Dashboard: dashboard, AcademicYear: newTestAcademicYearPageHandler(t), Class: newTestClassPageHandler(t), Transactions: handler}, MiddlewareSet{Authentication: auth, CSRF: middleware.NewCSRFMiddleware(security.NewDefaultCSRFCookie())}, newTestStaticAssetHandler())
	if err != nil {
		t.Fatal(err)
	}
	txID := uuid.New()
	form := url.Values{"amount": {"12345"}, "transaction_date": {"2026-10-20"}, "reason": {"Koreksi data"}}
	missingCSRF := httptest.NewRequest(http.MethodPost, "/transactions/"+txID.String()+"/edit", strings.NewReader(form.Encode()))
	missingCSRF.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	missingCSRF.AddCookie(&http.Cookie{Name: security.DefaultSessionCookieName, Value: sessionToken})
	missingRecorder := httptest.NewRecorder()
	router.ServeHTTP(missingRecorder, missingCSRF)
	if missingRecorder.Code != http.StatusForbidden || tx.editCalls != 0 {
		t.Fatalf("missing CSRF status/calls=%d/%d", missingRecorder.Code, tx.editCalls)
	}
	csrfToken := "valid-csrf-" + uuid.NewString()
	form.Set("csrf_token", csrfToken)
	post := httptest.NewRequest(http.MethodPost, "/transactions/"+txID.String()+"/edit", strings.NewReader(form.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.AddCookie(&http.Cookie{Name: security.DefaultSessionCookieName, Value: sessionToken})
	post.AddCookie(&http.Cookie{Name: security.DefaultCSRFCookieName, Value: csrfToken})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, post)
	if recorder.Code != http.StatusSeeOther || tx.editCalls != 1 || tx.editInput.TransactionID != txID || tx.editInput.Amount != 12_345 || tx.editInput.Reason != "Koreksi data" {
		t.Fatalf("edit status/calls/input=%d/%d/%+v", recorder.Code, tx.editCalls, tx.editInput)
	}
}

func (f *transactionPageAccountsFake) Get(_ context.Context, in service.GetSavingsAccountInput) (service.GetSavingsAccountOutput, error) {
	f.getInput = in
	return service.GetSavingsAccountOutput{Account: &f.account}, nil
}
func (f *transactionPageAccountsFake) GetBalance(context.Context, service.GetSavingsAccountBalanceInput) (service.GetSavingsAccountBalanceOutput, error) {
	return service.GetSavingsAccountBalanceOutput{Balance: 42_000}, nil
}

type transactionPageStudentsFake struct {
	service.StudentService
	student domain.Student
}

func (f *transactionPageStudentsFake) Get(_ context.Context, in service.GetStudentInput) (service.GetStudentOutput, error) {
	return service.GetStudentOutput{Student: &f.student}, nil
}

type transactionPageYearsFake struct {
	service.AcademicYearService
	year domain.AcademicYear
}

func (f *transactionPageYearsFake) Get(_ context.Context, in service.GetAcademicYearInput) (service.GetAcademicYearOutput, error) {
	return service.GetAcademicYearOutput{AcademicYear: &f.year}, nil
}

func newTransactionPageHandlerForTest(t *testing.T, tx *transactionPageServiceFake, account domain.SavingsAccount) *TransactionPageHandler {
	t.Helper()
	student := domain.Student{ID: account.StudentID, Name: "Ani", Status: domain.StudentActive}
	year := domain.AcademicYear{ID: account.AcademicYearID, Name: "2026/2027", Status: domain.AcademicYearOpen}
	studentUserID := uuid.New()
	student.UserID = &studentUserID
	accounts := &transactionPageAccountsFake{account: account}
	students := &transactionPageStudentsFake{student: student}
	years := &transactionPageYearsFake{year: year}
	tmpl := template.Must(template.New("transactions").Parse(`{{define "transactions"}}{{.StudentName}}|{{.AccountBalance}}{{range .Rows}}|{{.ID}}:{{.Balance}}{{end}}{{end}}`))
	h, err := NewTransactionPageHandler(tx, accounts, students, years, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestTransactionPageListUsesScopedServiceAndRunningBalance(t *testing.T) {
	account := domain.SavingsAccount{ID: uuid.New(), StudentID: uuid.New(), AcademicYearID: uuid.New(), Status: domain.SavingsAccountOpen}
	tx := domain.Transaction{ID: uuid.New(), SavingsAccountID: account.ID, Type: domain.TransactionDeposit, Amount: 42_000, TransactionDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}
	fake := &transactionPageServiceFake{listOutput: service.ListTransactionsOutput{
		Transactions: []*domain.Transaction{&tx}, RunningBalances: map[uuid.UUID]domain.Money{tx.ID: 42_000}, Balance: 42_000, Total: 1,
	}}
	h := newTransactionPageHandlerForTest(t, fake, account)
	actor := service.Actor{UserID: uuid.New(), Role: domain.RoleWaliKelas}
	req := requestWithActor(http.MethodGet, "/savings/"+account.ID.String()+"/transactions?page=1", actor)
	req.SetPathValue("account_id", account.ID.String())
	rec := httptest.NewRecorder()
	withCSRFForTest(http.HandlerFunc(h.List)).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || fake.listInput.SavingsAccountID != account.ID || !strings.Contains(rec.Body.String(), "Rp. 42.000") || !strings.Contains(rec.Body.String(), tx.ID.String()) {
		t.Fatalf("List status/body/input=%d/%s/%+v", rec.Code, rec.Body.String(), fake.listInput)
	}
}

func TestTransactionPageDepositParsesFormAndRedirects(t *testing.T) {
	account := domain.SavingsAccount{ID: uuid.New(), StudentID: uuid.New(), AcademicYearID: uuid.New(), Status: domain.SavingsAccountOpen}
	fake := &transactionPageServiceFake{}
	h := newTransactionPageHandlerForTest(t, fake, account)
	actor := service.Actor{UserID: uuid.New(), Role: domain.RoleWaliKelas}
	form := url.Values{"amount": {"25000"}, "transaction_date": {"2026-10-20"}}
	req := httptest.NewRequest(http.MethodPost, "/savings/"+account.ID.String()+"/transactions/deposits", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.SetPathValue("account_id", account.ID.String())
	req = req.WithContext(service.WithActor(req.Context(), actor))
	rec := httptest.NewRecorder()
	h.Deposit(rec, req)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/savings/"+account.ID.String()+"?notice=deposit_created" || fake.depositCalls != 1 || fake.depositInput.Actor != actor || fake.depositInput.Amount != 25_000 || fake.depositInput.TransactionDate.Format("2006-01-02") != "2026-10-20" {
		t.Fatalf("Deposit status/location/input/calls=%d/%q/%+v/%d", rec.Code, rec.Header().Get("Location"), fake.depositInput, fake.depositCalls)
	}
}

func TestTransactionPageRejectsAdminFinancialWrite(t *testing.T) {
	account := domain.SavingsAccount{ID: uuid.New(), StudentID: uuid.New(), AcademicYearID: uuid.New(), Status: domain.SavingsAccountOpen}
	fake := &transactionPageServiceFake{}
	h := newTransactionPageHandlerForTest(t, fake, account)
	form := url.Values{"amount": {"1000"}, "transaction_date": {"2026-10-20"}}
	req := httptest.NewRequest(http.MethodPost, "/savings/"+account.ID.String()+"/transactions/deposits", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("account_id", account.ID.String())
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}))
	rec := httptest.NewRecorder()
	h.Deposit(rec, req)
	if rec.Code != http.StatusForbidden || fake.depositCalls != 0 {
		t.Fatalf("Admin deposit status/calls=%d/%d, want 403/0", rec.Code, fake.depositCalls)
	}
}

func TestTransactionFormValidationRejectsStudentBeforeParsing(t *testing.T) {
	called := false
	h := validateTransactionRequestForm(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader("malformed"))
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Role: domain.RoleSiswa}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || called {
		t.Fatalf("middleware status/called=%d/%v, want forbidden/false", rec.Code, called)
	}
}

func TestTransactionFormValidationRejectsAdminBeforeParsing(t *testing.T) {
	called := false
	h := validateTransactionRequestForm(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader("malformed"))
	req = req.WithContext(service.WithActor(req.Context(), service.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || called {
		t.Fatalf("middleware status/called=%d/%v, want forbidden/false", rec.Code, called)
	}
}

func TestTransactionRoutesRequireAuthenticationCSRFAndUseScopedPaths(t *testing.T) {
	account := domain.SavingsAccount{ID: uuid.New(), StudentID: uuid.New(), AcademicYearID: uuid.New(), Status: domain.SavingsAccountOpen}
	tx := &transactionPageServiceFake{}
	transactions := newTransactionPageHandlerForTest(t, tx, account)
	userID := uuid.New()
	sessionToken := "transaction-session-" + uuid.NewString()
	session := domain.Session{ID: uuid.New(), UserID: userID, TokenHash: security.HashSessionToken(sessionToken), ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now()}
	user := domain.User{ID: userID, Username: "transaction-test", Role: domain.RoleWaliKelas}
	auth, err := middleware.NewAuthenticationMiddleware(security.NewDefaultSessionCookie(), &routerSessionRepositoryFake{session: session}, &routerUserRepositoryFake{user: user})
	if err != nil {
		t.Fatal(err)
	}
	templates := template.Must(template.New("pages").Parse(`{{define "transactions"}}{{.StudentName}}{{end}}`))
	dashboard, err := NewDashboardPageHandler(templates)
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewRouter(HandlerSet{
		Authentication: &AuthenticationHandler{}, LoginPage: newTestLoginPageHandler(t), Dashboard: dashboard,
		AcademicYear: newTestAcademicYearPageHandler(t), Class: newTestClassPageHandler(t), Transactions: transactions,
	}, MiddlewareSet{Authentication: auth, CSRF: middleware.NewCSRFMiddleware(security.NewDefaultCSRFCookie())}, newTestStaticAssetHandler())
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	unauthenticated := httptest.NewRecorder()
	router.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/savings/"+account.ID.String()+"/transactions", nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated list status=%d", unauthenticated.Code)
	}
	get := httptest.NewRequest(http.MethodGet, "/savings/"+account.ID.String()+"/transactions", nil)
	get.AddCookie(&http.Cookie{Name: security.DefaultSessionCookieName, Value: sessionToken})
	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, get)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("authenticated list status/body=%d/%s", getRecorder.Code, getRecorder.Body.String())
	}
	var csrfToken string
	for _, cookie := range getRecorder.Result().Cookies() {
		if cookie.Name == security.DefaultCSRFCookieName {
			csrfToken = cookie.Value
		}
	}
	if csrfToken == "" {
		t.Fatal("GET did not set the CSRF cookie")
	}
	form := url.Values{"amount": {"12345"}, "transaction_date": {"2026-10-20"}, "csrf_token": {csrfToken}}
	post := httptest.NewRequest(http.MethodPost, "/savings/"+account.ID.String()+"/transactions/deposits", strings.NewReader(form.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.AddCookie(&http.Cookie{Name: security.DefaultSessionCookieName, Value: sessionToken})
	post.AddCookie(&http.Cookie{Name: security.DefaultCSRFCookieName, Value: csrfToken})
	postRecorder := httptest.NewRecorder()
	router.ServeHTTP(postRecorder, post)
	if postRecorder.Code != http.StatusSeeOther || tx.depositCalls != 1 || tx.depositInput.Actor.Role != domain.RoleWaliKelas {
		t.Fatalf("authenticated POST status/calls/role=%d/%d/%s body=%s", postRecorder.Code, tx.depositCalls, tx.depositInput.Actor.Role, postRecorder.Body.String())
	}
}

func TestTransactionPostRejectsMalformedAndOversizedForms(t *testing.T) {
	account := domain.SavingsAccount{ID: uuid.New(), StudentID: uuid.New(), AcademicYearID: uuid.New(), Status: domain.SavingsAccountOpen}
	actor := service.Actor{UserID: uuid.New(), Role: domain.RoleWaliKelas}
	for _, tc := range []struct {
		name, contentType, body string
	}{
		{name: "missing content type", body: "amount=1&transaction_date=2026-10-20"},
		{name: "unsupported field", contentType: "application/x-www-form-urlencoded", body: "amount=1&transaction_date=2026-10-20&account_id=" + uuid.NewString()},
		{name: "reason only accepted for correction", contentType: "application/x-www-form-urlencoded", body: "amount=1&transaction_date=2026-10-20&reason=extra"},
		{name: "duplicate amount", contentType: "application/x-www-form-urlencoded", body: "amount=1&amount=2&transaction_date=2026-10-20"},
		{name: "malformed encoding", contentType: "application/x-www-form-urlencoded", body: "amount=1&transaction_date=2026-10-20&x=%zz"},
		{name: "oversized", contentType: "application/x-www-form-urlencoded", body: "amount=1&transaction_date=2026-10-20&x=" + strings.Repeat("a", maxTransactionFormBytes)},
		{name: "fractional amount", contentType: "application/x-www-form-urlencoded", body: "amount=1.5&transaction_date=2026-10-20"},
		{name: "invalid date", contentType: "application/x-www-form-urlencoded", body: "amount=1&transaction_date=2026-02-30"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &transactionPageServiceFake{}
			h := newTransactionPageHandlerForTest(t, fake, account)
			req := httptest.NewRequest(http.MethodPost, "/savings/"+account.ID.String()+"/transactions/deposits", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			req.SetPathValue("account_id", account.ID.String())
			req = req.WithContext(service.WithActor(req.Context(), actor))
			rec := httptest.NewRecorder()
			h.Deposit(rec, req)
			if rec.Code != http.StatusBadRequest || fake.depositCalls != 0 {
				t.Fatalf("Deposit status/calls=%d/%d, want 400/0", rec.Code, fake.depositCalls)
			}
		})
	}
}

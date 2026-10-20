package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type transactionServiceFakeSavingsAccountRepository struct {
	repository.SavingsAccountRepository

	getByIDFn             func(context.Context, uuid.UUID) (domain.SavingsAccount, error)
	getByIDForUpdateFn    func(context.Context, uuid.UUID) (domain.SavingsAccount, error)
	getByIDCalls          int
	getByIDForUpdateCalls int
}

func (f *transactionServiceFakeSavingsAccountRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.SavingsAccount, error) {
	f.getByIDCalls++
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	if f.getByIDForUpdateFn != nil {
		return f.getByIDForUpdateFn(ctx, id)
	}
	return domain.SavingsAccount{}, repository.ErrNotFound
}

func (f *transactionServiceFakeSavingsAccountRepository) GetByIDForUpdate(
	ctx context.Context,
	id uuid.UUID,
) (domain.SavingsAccount, error) {
	f.getByIDForUpdateCalls++
	if f.getByIDForUpdateFn != nil {
		return f.getByIDForUpdateFn(ctx, id)
	}
	return domain.SavingsAccount{}, repository.ErrNotFound
}

type transactionServiceFakeTransactionRepository struct {
	repository.TransactionRepository

	getByIDFn                    func(context.Context, uuid.UUID) (domain.Transaction, error)
	listBySavingsAccountFn       func(context.Context, uuid.UUID, repository.ListOptions) ([]domain.Transaction, error)
	listActiveBySavingsAccountFn func(context.Context, uuid.UUID) ([]domain.Transaction, error)
	createFn                     func(context.Context, domain.Transaction) error
	updateFn                     func(context.Context, domain.Transaction) error

	getByIDCalls    int
	listCalls       int
	listActiveCalls int
	createCalls     int
	updateCalls     int
	lastCreated     domain.Transaction
	lastUpdated     domain.Transaction
	lastListOptions repository.ListOptions
}

func (f *transactionServiceFakeTransactionRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.Transaction, error) {
	f.getByIDCalls++
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return domain.Transaction{}, repository.ErrNotFound
}

func (f *transactionServiceFakeTransactionRepository) ListBySavingsAccount(
	ctx context.Context,
	id uuid.UUID,
	options repository.ListOptions,
) ([]domain.Transaction, error) {
	f.listCalls++
	f.lastListOptions = options
	if f.listBySavingsAccountFn != nil {
		return f.listBySavingsAccountFn(ctx, id, options)
	}
	return nil, nil
}

func (f *transactionServiceFakeTransactionRepository) ListActiveBySavingsAccount(
	ctx context.Context,
	id uuid.UUID,
) ([]domain.Transaction, error) {
	f.listActiveCalls++
	if f.listActiveBySavingsAccountFn != nil {
		return f.listActiveBySavingsAccountFn(ctx, id)
	}
	return nil, nil
}

func (f *transactionServiceFakeTransactionRepository) Create(
	ctx context.Context,
	transaction domain.Transaction,
) error {
	f.createCalls++
	f.lastCreated = transaction
	if f.createFn != nil {
		return f.createFn(ctx, transaction)
	}
	return nil
}

func (f *transactionServiceFakeTransactionRepository) Update(
	ctx context.Context,
	transaction domain.Transaction,
) error {
	f.updateCalls++
	f.lastUpdated = transaction
	if f.updateFn != nil {
		return f.updateFn(ctx, transaction)
	}
	return nil
}

type transactionServiceFakeStudentRepository struct {
	repository.StudentRepository
	getByIDFn func(context.Context, uuid.UUID) (domain.Student, error)
}

func (f *transactionServiceFakeStudentRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.Student, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return domain.Student{}, repository.ErrNotFound
}

type transactionServiceFakeStudentClassHistoryRepository struct {
	repository.StudentClassHistoryRepository
	listFn func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.StudentClassHistory, error)
}

func (f *transactionServiceFakeStudentClassHistoryRepository) ListByStudentAndAcademicYear(
	ctx context.Context,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.StudentClassHistory, error) {
	if f.listFn != nil {
		return f.listFn(ctx, studentID, academicYearID, options)
	}
	return nil, nil
}

type transactionServiceFakeTeacherAssignmentRepository struct {
	repository.TeacherClassAssignmentRepository
	listByClassFn func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.TeacherClassAssignment, error)
}

func (f *transactionServiceFakeTeacherAssignmentRepository) ListByClassAndAcademicYear(
	ctx context.Context,
	classID uuid.UUID,
	academicYearID uuid.UUID,
	options repository.ListOptions,
) ([]domain.TeacherClassAssignment, error) {
	if f.listByClassFn != nil {
		return f.listByClassFn(ctx, classID, academicYearID, options)
	}
	return nil, nil
}

type transactionServiceFakeAuditRepository struct {
	repository.AuditRepository
	createFn    func(context.Context, domain.AuditLog) error
	createCalls int
	lastAudit   domain.AuditLog
}

type transactionServiceFakeAcademicYearRepository struct {
	repository.AcademicYearRepository
	getByIDForShareFn    func(context.Context, uuid.UUID) (domain.AcademicYear, error)
	getByIDForShareCalls int
}

func (f *transactionServiceFakeAcademicYearRepository) GetByIDForShare(
	ctx context.Context,
	id uuid.UUID,
) (domain.AcademicYear, error) {
	f.getByIDForShareCalls++
	if f.getByIDForShareFn != nil {
		return f.getByIDForShareFn(ctx, id)
	}
	return domain.AcademicYear{
		ID: id, StartDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate: time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC), Status: domain.AcademicYearOpen,
	}, nil
}

func (f *transactionServiceFakeAuditRepository) Create(
	ctx context.Context,
	auditLog domain.AuditLog,
) error {
	f.createCalls++
	f.lastAudit = auditLog
	if f.createFn != nil {
		return f.createFn(ctx, auditLog)
	}
	return nil
}

type transactionServiceFakeUnitOfWork struct {
	repos         repository.RepositorySet
	commitErr     error
	rollbackErr   error
	commitCalls   int
	rollbackCalls int
}

func (f *transactionServiceFakeUnitOfWork) Repositories() repository.RepositorySet {
	return f.repos
}

func (f *transactionServiceFakeUnitOfWork) Commit() error {
	f.commitCalls++
	return f.commitErr
}

func (f *transactionServiceFakeUnitOfWork) Rollback() error {
	f.rollbackCalls++
	return f.rollbackErr
}

type transactionServiceFakeUnitOfWorkManager struct {
	uow        repository.UnitOfWork
	beginErr   error
	beginCalls int
}

func (f *transactionServiceFakeUnitOfWorkManager) Begin(context.Context) (repository.UnitOfWork, error) {
	f.beginCalls++
	if f.beginErr != nil {
		return nil, f.beginErr
	}
	return f.uow, nil
}

func newTransactionServiceForTest(t *testing.T, repos repository.RepositorySet, manager repository.UnitOfWorkManager) *transactionService {
	t.Helper()
	svc, err := NewTransactionService(Dependencies{Repositories: repos, UOW: manager})
	if err != nil {
		t.Fatalf("NewTransactionService() error = %v", err)
	}
	return svc
}

func newTransactionTestAccount(t *testing.T) domain.SavingsAccount {
	t.Helper()
	account, err := domain.NewSavingsAccount(uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}
	return account
}

func newTransactionTestDeposit(t *testing.T, accountID uuid.UUID, amount domain.Money) domain.Transaction {
	t.Helper()
	transaction, err := domain.NewDeposit(accountID, amount, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), uuid.New())
	if err != nil {
		t.Fatalf("NewDeposit() error = %v", err)
	}
	return transaction
}

func newTransactionTestRepositories(
	accountRepo repository.SavingsAccountRepository,
	transactionRepo repository.TransactionRepository,
	studentRepo repository.StudentRepository,
	historyRepo repository.StudentClassHistoryRepository,
	assignmentRepo repository.TeacherClassAssignmentRepository,
	auditRepo repository.AuditRepository,
) repository.RepositorySet {
	return repository.RepositorySet{
		AcademicYears:           &transactionServiceFakeAcademicYearRepository{},
		SavingsAccounts:         accountRepo,
		Transactions:            transactionRepo,
		Students:                studentRepo,
		StudentClassHistories:   historyRepo,
		TeacherClassAssignments: assignmentRepo,
		AuditLogs:               auditRepo,
	}
}

func TestTransactionServiceDepositWaliKelas(t *testing.T) {
	account := newTransactionTestAccount(t)
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) {
			return account, nil
		},
	}
	transactionRepo := &transactionServiceFakeTransactionRepository{}
	auditRepo := &transactionServiceFakeAuditRepository{}
	uow := &transactionServiceFakeUnitOfWork{
		repos: newTransactionTestRepositories(accountRepo, transactionRepo, nil, nil, nil, auditRepo),
	}
	manager := &transactionServiceFakeUnitOfWorkManager{uow: uow}
	svc := newScopedTransactionServiceForTest(t, uow.repos, manager)
	actor := Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}

	got, err := svc.Deposit(context.Background(), DepositInput{
		Actor: actor, SavingsAccountID: account.ID, Amount: 100_000,
		TransactionDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Deposit() error = %v", err)
	}
	if got.Transaction == nil || got.Transaction.Type != domain.TransactionDeposit {
		t.Fatalf("Deposit() output = %#v, want deposit transaction", got.Transaction)
	}
	if accountRepo.getByIDForUpdateCalls != 1 || transactionRepo.createCalls != 1 || auditRepo.createCalls != 1 {
		t.Fatalf("calls = account lock %d, create %d, audit %d; want 1, 1, 1", accountRepo.getByIDForUpdateCalls, transactionRepo.createCalls, auditRepo.createCalls)
	}
	if uow.commitCalls != 1 || uow.rollbackCalls != 0 {
		t.Fatalf("uow calls = commit %d rollback %d; want 1 0", uow.commitCalls, uow.rollbackCalls)
	}
	if auditRepo.lastAudit.Action != "CREATE" || auditRepo.lastAudit.EntityType != domain.AuditEntityTransaction {
		t.Fatalf("audit = %#v, want CREATE TRANSACTION", auditRepo.lastAudit)
	}
	if accountRepo.getByIDCalls != 1 || accountRepo.getByIDForUpdateCalls != 1 {
		t.Fatalf("account reads/unlocked-lock reads = %d/%d, want 1/1", accountRepo.getByIDCalls, accountRepo.getByIDForUpdateCalls)
	}
	yearRepo := uow.repos.AcademicYears.(*transactionServiceFakeAcademicYearRepository)
	if yearRepo.getByIDForShareCalls != 1 {
		t.Fatalf("year shared locks = %d, want 1", yearRepo.getByIDForShareCalls)
	}
}

func TestTransactionServiceDepositRequiresOpenYearAndInRangeDate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		date    time.Time
		status  domain.AcademicYearStatus
		wantErr error
	}{
		{name: "start inclusive", date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), status: domain.AcademicYearOpen},
		{name: "end exclusive", date: time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC), status: domain.AcademicYearOpen, wantErr: ErrTransactionDateOutOfAcademicYear},
		{name: "year closing", date: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), status: domain.AcademicYearClosing, wantErr: domain.ErrInvalidTransition},
	} {
		t.Run(tc.name, func(t *testing.T) {
			account := newTransactionTestAccount(t)
			accountRepo := &transactionServiceFakeSavingsAccountRepository{
				getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
			}
			yearRepo := &transactionServiceFakeAcademicYearRepository{getByIDForShareFn: func(_ context.Context, id uuid.UUID) (domain.AcademicYear, error) {
				return domain.AcademicYear{ID: id, StartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC), Status: tc.status}, nil
			}}
			repos := newTransactionTestRepositories(accountRepo, &transactionServiceFakeTransactionRepository{}, nil, nil, nil, &transactionServiceFakeAuditRepository{})
			repos.AcademicYears = yearRepo
			uow := &transactionServiceFakeUnitOfWork{repos: repos}
			svc := newScopedTransactionServiceForTest(t, repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})
			_, err := svc.Deposit(context.Background(), DepositInput{
				Actor: Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}, SavingsAccountID: account.ID,
				Amount: 10_000, TransactionDate: tc.date,
			})
			if tc.wantErr == nil && err != nil || tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("Deposit() error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil && (accountRepo.getByIDForUpdateCalls != 0 || uow.commitCalls != 0) {
				t.Fatalf("rejected date/year proceeded to account write lock/commit: %d/%d", accountRepo.getByIDForUpdateCalls, uow.commitCalls)
			}
		})
	}
}

func TestTransactionServiceDepositRejectsInvalidHistoricalBalance(t *testing.T) {
	account := newTransactionTestAccount(t)
	withdrawal, err := domain.NewWithdrawal(account.ID, 100_000, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), uuid.New())
	if err != nil {
		t.Fatalf("NewWithdrawal() error = %v", err)
	}
	transactionRepo := &transactionServiceFakeTransactionRepository{
		listActiveBySavingsAccountFn: func(context.Context, uuid.UUID) ([]domain.Transaction, error) {
			return []domain.Transaction{withdrawal}, nil
		},
	}
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}
	uow := &transactionServiceFakeUnitOfWork{repos: newTransactionTestRepositories(accountRepo, transactionRepo, nil, nil, nil, &transactionServiceFakeAuditRepository{})}
	svc := newScopedTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	_, err = svc.Deposit(context.Background(), DepositInput{Actor: Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}, SavingsAccountID: account.ID, Amount: 200_000, TransactionDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)})
	if !errors.Is(err, domain.ErrNegativeBalance) {
		t.Fatalf("Deposit() error = %v, want %v", err, domain.ErrNegativeBalance)
	}
	if transactionRepo.createCalls != 0 || uow.commitCalls != 0 || uow.rollbackCalls != 1 {
		t.Fatalf("create/commit/rollback = %d/%d/%d, want 0/0/1", transactionRepo.createCalls, uow.commitCalls, uow.rollbackCalls)
	}
}

func TestTransactionServiceDepositReusesOuterTransaction(t *testing.T) {
	account := newTransactionTestAccount(t)
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) {
			return account, nil
		},
	}
	transactionRepo := &transactionServiceFakeTransactionRepository{}
	auditRepo := &transactionServiceFakeAuditRepository{}
	uow := &transactionServiceFakeUnitOfWork{
		repos: newTransactionTestRepositories(accountRepo, transactionRepo, nil, nil, nil, auditRepo),
	}
	manager := &transactionServiceFakeUnitOfWorkManager{uow: uow}
	svc := newScopedTransactionServiceForTest(t, uow.repos, manager)

	err := WithinTransaction(
		context.Background(),
		manager,
		func(ctx context.Context, _ repository.RepositorySet) error {
			_, err := svc.Deposit(ctx, DepositInput{
				Actor:            Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas},
				SavingsAccountID: account.ID,
				Amount:           100_000,
				TransactionDate:  time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
			})
			return err
		},
	)
	if err != nil {
		t.Fatalf("WithinTransaction() error = %v", err)
	}

	if manager.beginCalls != 1 {
		t.Fatalf("Begin calls = %d, want 1", manager.beginCalls)
	}
	if uow.commitCalls != 1 {
		t.Fatalf("Commit calls = %d, want 1", uow.commitCalls)
	}
	if uow.rollbackCalls != 0 {
		t.Fatalf("Rollback calls = %d, want 0", uow.rollbackCalls)
	}
	if transactionRepo.createCalls != 1 || auditRepo.createCalls != 1 {
		t.Fatalf("calls = transaction create %d audit create %d, want 1 1", transactionRepo.createCalls, auditRepo.createCalls)
	}
}

func TestTransactionServiceWithdrawalRejectsInsufficientBalance(t *testing.T) {
	account := newTransactionTestAccount(t)
	transactionRepo := &transactionServiceFakeTransactionRepository{
		listActiveBySavingsAccountFn: func(context.Context, uuid.UUID) ([]domain.Transaction, error) {
			return []domain.Transaction{newTransactionTestDeposit(t, account.ID, 50_000)}, nil
		},
	}
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) {
			return account, nil
		},
	}
	auditRepo := &transactionServiceFakeAuditRepository{}
	uow := &transactionServiceFakeUnitOfWork{repos: newTransactionTestRepositories(accountRepo, transactionRepo, nil, nil, nil, auditRepo)}
	svc := newScopedTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.Withdrawal(context.Background(), WithdrawalInput{
		Actor:            Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas},
		SavingsAccountID: account.ID, Amount: 75_000,
		TransactionDate: time.Now(),
	})
	if !errors.Is(err, domain.ErrNegativeBalance) {
		t.Fatalf("Withdrawal() error = %v, want %v", err, domain.ErrNegativeBalance)
	}
	if accountRepo.getByIDForUpdateCalls != 1 || transactionRepo.createCalls != 0 || auditRepo.createCalls != 0 {
		t.Fatalf("calls = lock %d create %d audit %d; want 1 0 0", accountRepo.getByIDForUpdateCalls, transactionRepo.createCalls, auditRepo.createCalls)
	}
	if uow.commitCalls != 0 || uow.rollbackCalls != 1 {
		t.Fatalf("uow calls = commit %d rollback %d; want 0 1", uow.commitCalls, uow.rollbackCalls)
	}
}

func TestTransactionServiceWithdrawalUsesRowLockAndCommits(t *testing.T) {
	account := newTransactionTestAccount(t)
	transactionRepo := &transactionServiceFakeTransactionRepository{
		listActiveBySavingsAccountFn: func(context.Context, uuid.UUID) ([]domain.Transaction, error) {
			return []domain.Transaction{newTransactionTestDeposit(t, account.ID, 100_000)}, nil
		},
	}
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) {
			return account, nil
		},
	}
	auditRepo := &transactionServiceFakeAuditRepository{}
	uow := &transactionServiceFakeUnitOfWork{repos: newTransactionTestRepositories(accountRepo, transactionRepo, nil, nil, nil, auditRepo)}
	svc := newScopedTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	got, err := svc.Withdrawal(context.Background(), WithdrawalInput{
		Actor:            Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas},
		SavingsAccountID: account.ID, Amount: 75_000,
		TransactionDate: time.Now(),
	})
	if err != nil {
		t.Fatalf("Withdrawal() error = %v", err)
	}
	if got.Transaction == nil || got.Transaction.Type != domain.TransactionWithdrawal {
		t.Fatalf("Withdrawal() output = %#v, want withdrawal", got.Transaction)
	}
	if accountRepo.getByIDForUpdateCalls != 1 || transactionRepo.listActiveCalls != 1 || transactionRepo.createCalls != 1 {
		t.Fatalf("calls = lock %d active-list %d create %d; want 1 1 1", accountRepo.getByIDForUpdateCalls, transactionRepo.listActiveCalls, transactionRepo.createCalls)
	}
	if uow.commitCalls != 1 || uow.rollbackCalls != 0 {
		t.Fatalf("uow calls = commit %d rollback %d; want 1 0", uow.commitCalls, uow.rollbackCalls)
	}
}

func TestTransactionServiceGetStudentSelf(t *testing.T) {
	account := newTransactionTestAccount(t)
	transaction := newTransactionTestDeposit(t, account.ID, 100_000)
	studentUserID := uuid.New()
	transactionRepo := &transactionServiceFakeTransactionRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Transaction, error) { return transaction, nil },
	}
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}
	studentRepo := &transactionServiceFakeStudentRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Student, error) {
			return domain.Student{ID: account.StudentID, UserID: &studentUserID}, nil
		},
	}
	svc := newTransactionServiceForTest(t,
		newTransactionTestRepositories(accountRepo, transactionRepo, studentRepo, nil, nil, nil),
		&transactionServiceFakeUnitOfWorkManager{},
	)

	got, err := svc.Get(context.Background(), GetTransactionInput{
		Actor: Actor{UserID: studentUserID, Role: domain.RoleSiswa}, TransactionID: transaction.ID,
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Transaction == nil || got.Transaction.ID != transaction.ID {
		t.Fatalf("Get() output = %#v, want transaction %v", got.Transaction, transaction.ID)
	}
}

func TestTransactionServiceGetStudentOtherUserRejected(t *testing.T) {
	account := newTransactionTestAccount(t)
	transaction := newTransactionTestDeposit(t, account.ID, 100_000)
	studentUserID := uuid.New()
	transactionRepo := &transactionServiceFakeTransactionRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Transaction, error) { return transaction, nil },
	}
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}
	studentRepo := &transactionServiceFakeStudentRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Student, error) {
			return domain.Student{ID: account.StudentID, UserID: &studentUserID}, nil
		},
	}
	svc := newTransactionServiceForTest(t,
		newTransactionTestRepositories(accountRepo, transactionRepo, studentRepo, nil, nil, nil),
		&transactionServiceFakeUnitOfWorkManager{},
	)

	_, err := svc.Get(context.Background(), GetTransactionInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleSiswa}, TransactionID: transaction.ID,
	})
	if !errors.Is(err, ErrScopeViolation) {
		t.Fatalf("Get() error = %v, want %v", err, ErrScopeViolation)
	}
}

func TestTransactionServiceListPassesOptions(t *testing.T) {
	account := newTransactionTestAccount(t)
	transaction := newTransactionTestDeposit(t, account.ID, 100_000)
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}
	transactionRepo := &transactionServiceFakeTransactionRepository{
		listActiveBySavingsAccountFn: func(context.Context, uuid.UUID) ([]domain.Transaction, error) {
			return []domain.Transaction{transaction}, nil
		},
	}
	svc := newTransactionServiceForTest(t,
		newTransactionTestRepositories(accountRepo, transactionRepo, nil, nil, nil, nil),
		&transactionServiceFakeUnitOfWorkManager{},
	)

	got, err := svc.List(context.Background(), ListTransactionsInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, SavingsAccountID: account.ID,
		Options: ListOptions{Limit: 10, Offset: 0},
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got.Transactions) != 1 || got.Transactions[0].ID != transaction.ID {
		t.Fatalf("List() output = %#v, want one transaction", got.Transactions)
	}
	if got.Total != 1 || len(got.RunningBalances) != 1 || got.RunningBalances[transaction.ID] != 100_000 {
		t.Fatalf("List() totals/balances = %d/%v", got.Total, got.RunningBalances)
	}
}

func TestTransactionServiceListKeepsLedgerBalanceAcrossPages(t *testing.T) {
	account := newTransactionTestAccount(t)
	created := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	deposit1 := domain.Transaction{ID: uuid.New(), SavingsAccountID: account.ID, Type: domain.TransactionDeposit, Amount: 100_000, Status: domain.TransactionActive, TransactionDate: created, CreatedAt: created}
	withdrawal := domain.Transaction{ID: uuid.New(), SavingsAccountID: account.ID, Type: domain.TransactionWithdrawal, Amount: 20_000, Status: domain.TransactionActive, TransactionDate: created.AddDate(0, 0, 1), CreatedAt: created}
	deposit2 := domain.Transaction{ID: uuid.New(), SavingsAccountID: account.ID, Type: domain.TransactionDeposit, Amount: 50_000, Status: domain.TransactionActive, TransactionDate: created.AddDate(0, 0, 2), CreatedAt: created}
	accountRepo := &transactionServiceFakeSavingsAccountRepository{getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil }}
	txRepo := &transactionServiceFakeTransactionRepository{listActiveBySavingsAccountFn: func(context.Context, uuid.UUID) ([]domain.Transaction, error) {
		return []domain.Transaction{deposit2, withdrawal, deposit1}, nil
	}}
	svc := newTransactionServiceForTest(t, newTransactionTestRepositories(accountRepo, txRepo, nil, nil, nil, nil), &transactionServiceFakeUnitOfWorkManager{})
	out, err := svc.List(context.Background(), ListTransactionsInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, SavingsAccountID: account.ID,
		Options: ListOptions{Limit: 1, Offset: 1},
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if out.Total != 3 || out.Balance != 130_000 || len(out.Transactions) != 1 || out.Transactions[0].ID != withdrawal.ID || out.RunningBalances[withdrawal.ID] != 80_000 {
		t.Fatalf("List() output = total %d final %d rows %+v balances %v", out.Total, out.Balance, out.Transactions, out.RunningBalances)
	}
}

func TestTransactionServiceEditRecalculatesBalanceAndAudits(t *testing.T) {
	account := newTransactionTestAccount(t)
	original := newTransactionTestDeposit(t, account.ID, 100_000)
	current := original
	transactionRepo := &transactionServiceFakeTransactionRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Transaction, error) { return current, nil },
		listActiveBySavingsAccountFn: func(context.Context, uuid.UUID) ([]domain.Transaction, error) {
			return []domain.Transaction{current}, nil
		},
	}
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}
	auditRepo := &transactionServiceFakeAuditRepository{}
	uow := &transactionServiceFakeUnitOfWork{repos: newTransactionTestRepositories(accountRepo, transactionRepo, nil, nil, nil, auditRepo)}
	svc := newScopedTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	got, err := svc.Edit(context.Background(), EditTransactionInput{
		Actor: Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}, TransactionID: original.ID,
		Amount: 75_000, TransactionDate: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Edit() error = %v", err)
	}
	if got.Transaction.Amount != 75_000 || !got.Transaction.TransactionDate.Equal(time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("Edit() transaction = %#v, want amount/date updated", got.Transaction)
	}
	if transactionRepo.updateCalls != 1 || auditRepo.createCalls != 1 || accountRepo.getByIDForUpdateCalls != 1 {
		t.Fatalf("calls = update %d audit %d lock %d; want 1 1 1", transactionRepo.updateCalls, auditRepo.createCalls, accountRepo.getByIDForUpdateCalls)
	}
	if auditRepo.lastAudit.Action != "UPDATE" || len(auditRepo.lastAudit.BeforeData) == 0 || len(auditRepo.lastAudit.AfterData) == 0 {
		t.Fatalf("audit = %#v, want UPDATE with before/after", auditRepo.lastAudit)
	}
}

func TestTransactionServiceEditRejectsNonPositiveAmount(t *testing.T) {
	account := newTransactionTestAccount(t)
	original := newTransactionTestDeposit(t, account.ID, 100_000)
	transactionRepo := &transactionServiceFakeTransactionRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Transaction, error) { return original, nil },
	}
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}
	uow := &transactionServiceFakeUnitOfWork{repos: newTransactionTestRepositories(accountRepo, transactionRepo, nil, nil, nil, nil)}
	manager := &transactionServiceFakeUnitOfWorkManager{uow: uow}
	svc := newScopedTransactionServiceForTest(t, uow.repos, manager)
	_, err := svc.Edit(context.Background(), EditTransactionInput{Actor: Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}, TransactionID: original.ID, Amount: 0, TransactionDate: original.TransactionDate})
	if !errors.Is(err, domain.ErrInvalidAmount) {
		t.Fatalf("Edit() error = %v, want %v", err, domain.ErrInvalidAmount)
	}
	if manager.beginCalls != 0 || transactionRepo.updateCalls != 0 {
		t.Fatalf("begin/update calls = %d/%d, want 0/0", manager.beginCalls, transactionRepo.updateCalls)
	}
}

func TestTransactionServiceEditRejectsNegativeResultingBalance(t *testing.T) {
	account := newTransactionTestAccount(t)
	original := newTransactionTestDeposit(t, account.ID, 100_000)
	withdrawal := func() domain.Transaction {
		tx, err := domain.NewWithdrawal(account.ID, 90_000, time.Now(), uuid.New())
		if err != nil {
			t.Fatalf("NewWithdrawal() error = %v", err)
		}
		return tx
	}()
	current := original
	transactionRepo := &transactionServiceFakeTransactionRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Transaction, error) { return current, nil },
		listActiveBySavingsAccountFn: func(context.Context, uuid.UUID) ([]domain.Transaction, error) {
			return []domain.Transaction{current, withdrawal}, nil
		},
	}
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}
	uow := &transactionServiceFakeUnitOfWork{repos: newTransactionTestRepositories(accountRepo, transactionRepo, nil, nil, nil, nil)}
	svc := newScopedTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.Edit(context.Background(), EditTransactionInput{
		Actor: Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}, TransactionID: original.ID,
		Amount: 50_000, TransactionDate: time.Now(),
	})
	if !errors.Is(err, domain.ErrNegativeBalance) {
		t.Fatalf("Edit() error = %v, want %v", err, domain.ErrNegativeBalance)
	}
	if transactionRepo.updateCalls != 0 || uow.commitCalls != 0 || uow.rollbackCalls != 1 {
		t.Fatalf("calls = update %d commit %d rollback %d; want 0 0 1", transactionRepo.updateCalls, uow.commitCalls, uow.rollbackCalls)
	}
}

func TestTransactionServiceEditRejectsDateOutsideAcademicYear(t *testing.T) {
	account := newTransactionTestAccount(t)
	transaction := newTransactionTestDeposit(t, account.ID, 100_000)
	accountRepo := &transactionServiceFakeSavingsAccountRepository{getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil }}
	txRepo := &transactionServiceFakeTransactionRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Transaction, error) { return transaction, nil },
		listActiveBySavingsAccountFn: func(context.Context, uuid.UUID) ([]domain.Transaction, error) {
			return []domain.Transaction{transaction}, nil
		},
	}
	yearRepo := &transactionServiceFakeAcademicYearRepository{getByIDForShareFn: func(_ context.Context, id uuid.UUID) (domain.AcademicYear, error) {
		return domain.AcademicYear{ID: id, StartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC), Status: domain.AcademicYearOpen}, nil
	}}
	repos := newTransactionTestRepositories(accountRepo, txRepo, nil, nil, nil, &transactionServiceFakeAuditRepository{})
	repos.AcademicYears = yearRepo
	uow := &transactionServiceFakeUnitOfWork{repos: repos}
	svc := newScopedTransactionServiceForTest(t, repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})
	_, err := svc.Edit(context.Background(), EditTransactionInput{
		Actor:         Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas},
		TransactionID: transaction.ID, Amount: 50_000,
		TransactionDate: time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, ErrTransactionDateOutOfAcademicYear) {
		t.Fatalf("Edit() error = %v, want %v", err, ErrTransactionDateOutOfAcademicYear)
	}
	if accountRepo.getByIDForUpdateCalls != 0 || txRepo.updateCalls != 0 || uow.commitCalls != 0 {
		t.Fatalf("out-of-range edit proceeded: account locks=%d transaction updates=%d commits=%d", accountRepo.getByIDForUpdateCalls, txRepo.updateCalls, uow.commitCalls)
	}
}

func TestTransactionServiceCancelRejectsSettledAccount(t *testing.T) {
	account := newTransactionTestAccount(t)
	if err := account.Settle(time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("SavingsAccount.Settle() error = %v", err)
	}

	original := newTransactionTestDeposit(t, account.ID, 100_000)
	transactionRepo := &transactionServiceFakeTransactionRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Transaction, error) { return original, nil },
	}
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}
	auditRepo := &transactionServiceFakeAuditRepository{}
	uow := &transactionServiceFakeUnitOfWork{repos: newTransactionTestRepositories(accountRepo, transactionRepo, nil, nil, nil, auditRepo)}
	svc := newScopedTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.Cancel(context.Background(), CancelTransactionInput{
		Actor: Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}, TransactionID: original.ID,
	})
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("Cancel() error = %v, want %v", err, domain.ErrInvalidTransition)
	}
	if transactionRepo.updateCalls != 0 || auditRepo.createCalls != 0 || uow.commitCalls != 0 || uow.rollbackCalls != 1 {
		t.Fatalf("calls = update %d audit %d commit %d rollback %d; want 0 0 0 1", transactionRepo.updateCalls, auditRepo.createCalls, uow.commitCalls, uow.rollbackCalls)
	}
}

func TestTransactionServiceCancelRejectsInvalidHistoricalBalance(t *testing.T) {
	account := newTransactionTestAccount(t)
	deposit := newTransactionTestDeposit(t, account.ID, 100_000)
	withdrawal, err := domain.NewWithdrawal(account.ID, 90_000, deposit.TransactionDate.AddDate(0, 0, 1), uuid.New())
	if err != nil {
		t.Fatalf("NewWithdrawal() error = %v", err)
	}
	transactionRepo := &transactionServiceFakeTransactionRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Transaction, error) { return deposit, nil },
		listActiveBySavingsAccountFn: func(context.Context, uuid.UUID) ([]domain.Transaction, error) {
			return []domain.Transaction{deposit, withdrawal}, nil
		},
	}
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}
	uow := &transactionServiceFakeUnitOfWork{repos: newTransactionTestRepositories(accountRepo, transactionRepo, nil, nil, nil, &transactionServiceFakeAuditRepository{})}
	svc := newScopedTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})
	_, err = svc.Cancel(context.Background(), CancelTransactionInput{Actor: Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}, TransactionID: deposit.ID})
	if !errors.Is(err, domain.ErrNegativeBalance) {
		t.Fatalf("Cancel() error = %v, want %v", err, domain.ErrNegativeBalance)
	}
	if transactionRepo.updateCalls != 0 || uow.commitCalls != 0 || uow.rollbackCalls != 1 {
		t.Fatalf("update/commit/rollback = %d/%d/%d, want 0/0/1", transactionRepo.updateCalls, uow.commitCalls, uow.rollbackCalls)
	}
}

func TestTransactionServiceRejectsStudentMutation(t *testing.T) {
	service := newTransactionServiceForTest(t, repository.RepositorySet{}, &transactionServiceFakeUnitOfWorkManager{})
	actor := Actor{UserID: uuid.New(), Role: domain.RoleSiswa}

	if _, err := service.Deposit(context.Background(), DepositInput{Actor: actor, SavingsAccountID: uuid.New(), Amount: 1, TransactionDate: time.Now()}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Deposit() error = %v, want %v", err, ErrForbidden)
	}
	if _, err := service.Withdrawal(context.Background(), WithdrawalInput{Actor: actor, SavingsAccountID: uuid.New(), Amount: 1, TransactionDate: time.Now()}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Withdrawal() error = %v, want %v", err, ErrForbidden)
	}
	if _, err := service.Edit(context.Background(), EditTransactionInput{Actor: actor, TransactionID: uuid.New(), Amount: 1, TransactionDate: time.Now()}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Edit() error = %v, want %v", err, ErrForbidden)
	}
	if _, err := service.Cancel(context.Background(), CancelTransactionInput{Actor: actor, TransactionID: uuid.New()}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Cancel() error = %v, want %v", err, ErrForbidden)
	}
}

func TestTransactionServiceRollsBackOnAuditFailure(t *testing.T) {
	account := newTransactionTestAccount(t)
	transactionRepo := &transactionServiceFakeTransactionRepository{}
	auditRepo := &transactionServiceFakeAuditRepository{createFn: func(context.Context, domain.AuditLog) error {
		return errors.New("audit failure")
	}}
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}
	uow := &transactionServiceFakeUnitOfWork{repos: newTransactionTestRepositories(accountRepo, transactionRepo, nil, nil, nil, auditRepo)}
	svc := newScopedTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.Deposit(context.Background(), DepositInput{
		Actor: Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}, SavingsAccountID: account.ID,
		Amount: 100_000, TransactionDate: time.Now(),
	})
	if err == nil || err.Error() != "audit failure" {
		t.Fatalf("Deposit() error = %v, want audit failure", err)
	}
	if uow.commitCalls != 0 || uow.rollbackCalls != 1 {
		t.Fatalf("uow calls = commit %d rollback %d; want 0 1", uow.commitCalls, uow.rollbackCalls)
	}
}

func TestTransactionServiceBeginFailure(t *testing.T) {
	beginErr := errors.New("begin failure")
	svc := newScopedTransactionServiceForTest(t, repository.RepositorySet{}, &transactionServiceFakeUnitOfWorkManager{beginErr: beginErr})
	_, err := svc.Deposit(context.Background(), DepositInput{
		Actor: Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}, SavingsAccountID: uuid.New(), Amount: 1,
		TransactionDate: time.Now(),
	})
	if err == nil || !errors.Is(err, beginErr) {
		t.Fatalf("Deposit() error = %v, want wrapped begin failure", err)
	}
}

func TestTransactionServiceDepositWaliKelasInScope(t *testing.T) {
	account := newTransactionTestAccount(t)
	actorID := uuid.New()
	classID := uuid.New()
	account.StudentID = uuid.New()
	historyRepo := &transactionServiceFakeStudentClassHistoryRepository{
		listFn: func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.StudentClassHistory, error) {
			return []domain.StudentClassHistory{{ClassID: classID}}, nil
		},
	}
	assignmentRepo := &transactionServiceFakeTeacherAssignmentRepository{
		listByClassFn: func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.TeacherClassAssignment, error) {
			return []domain.TeacherClassAssignment{{UserID: actorID}}, nil
		},
	}
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}
	transactionRepo := &transactionServiceFakeTransactionRepository{}
	auditRepo := &transactionServiceFakeAuditRepository{}
	uow := &transactionServiceFakeUnitOfWork{
		repos: newTransactionTestRepositories(accountRepo, transactionRepo, nil, historyRepo, assignmentRepo, auditRepo),
	}
	svc := newTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.Deposit(context.Background(), DepositInput{
		Actor:            Actor{UserID: actorID, Role: domain.RoleWaliKelas},
		SavingsAccountID: account.ID,
		Amount:           100_000,
		TransactionDate:  time.Now(),
	})
	if err != nil {
		t.Fatalf("Deposit() error = %v", err)
	}
	if transactionRepo.createCalls != 1 || auditRepo.createCalls != 1 || uow.commitCalls != 1 {
		t.Fatalf("calls = create %d audit %d commit %d; want 1 1 1", transactionRepo.createCalls, auditRepo.createCalls, uow.commitCalls)
	}
}

func TestTransactionServiceDepositWaliKelasOutOfScope(t *testing.T) {
	account := newTransactionTestAccount(t)
	actorID := uuid.New()
	classID := uuid.New()
	historyRepo := &transactionServiceFakeStudentClassHistoryRepository{
		listFn: func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.StudentClassHistory, error) {
			return []domain.StudentClassHistory{{ClassID: classID}}, nil
		},
	}
	assignmentRepo := &transactionServiceFakeTeacherAssignmentRepository{
		listByClassFn: func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.TeacherClassAssignment, error) {
			return []domain.TeacherClassAssignment{{UserID: uuid.New()}}, nil
		},
	}
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}
	transactionRepo := &transactionServiceFakeTransactionRepository{}
	auditRepo := &transactionServiceFakeAuditRepository{}
	uow := &transactionServiceFakeUnitOfWork{
		repos: newTransactionTestRepositories(accountRepo, transactionRepo, nil, historyRepo, assignmentRepo, auditRepo),
	}
	svc := newTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.Deposit(context.Background(), DepositInput{
		Actor:            Actor{UserID: actorID, Role: domain.RoleWaliKelas},
		SavingsAccountID: account.ID,
		Amount:           100_000,
		TransactionDate:  time.Now(),
	})
	if !errors.Is(err, ErrScopeViolation) {
		t.Fatalf("Deposit() error = %v, want %v", err, ErrScopeViolation)
	}
	if transactionRepo.createCalls != 0 || auditRepo.createCalls != 0 || uow.commitCalls != 0 || uow.rollbackCalls != 1 {
		t.Fatalf("calls = create %d audit %d commit %d rollback %d; want 0 0 0 1", transactionRepo.createCalls, auditRepo.createCalls, uow.commitCalls, uow.rollbackCalls)
	}
}

func TestTransactionServiceCancelAlreadyCancelled(t *testing.T) {
	account := newTransactionTestAccount(t)
	transaction := newTransactionTestDeposit(t, account.ID, 100_000)
	if err := transaction.Cancel(); err != nil {
		t.Fatalf("Transaction.Cancel() error = %v", err)
	}
	transactionRepo := &transactionServiceFakeTransactionRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Transaction, error) { return transaction, nil },
	}
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}
	auditRepo := &transactionServiceFakeAuditRepository{}
	uow := &transactionServiceFakeUnitOfWork{repos: newTransactionTestRepositories(accountRepo, transactionRepo, nil, nil, nil, auditRepo)}
	svc := newScopedTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.Cancel(context.Background(), CancelTransactionInput{
		Actor: Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}, TransactionID: transaction.ID,
	})
	if !errors.Is(err, domain.ErrAlreadyCancelled) {
		t.Fatalf("Cancel() error = %v, want %v", err, domain.ErrAlreadyCancelled)
	}
	if transactionRepo.updateCalls != 0 || auditRepo.createCalls != 0 || uow.commitCalls != 0 || uow.rollbackCalls != 1 {
		t.Fatalf("calls = update %d audit %d commit %d rollback %d; want 0 0 0 1", transactionRepo.updateCalls, auditRepo.createCalls, uow.commitCalls, uow.rollbackCalls)
	}
}

func TestTransactionServiceCommitFailureRollsBack(t *testing.T) {
	account := newTransactionTestAccount(t)
	transactionRepo := &transactionServiceFakeTransactionRepository{}
	auditRepo := &transactionServiceFakeAuditRepository{}
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}
	commitErr := errors.New("commit failure")
	uow := &transactionServiceFakeUnitOfWork{
		repos:     newTransactionTestRepositories(accountRepo, transactionRepo, nil, nil, nil, auditRepo),
		commitErr: commitErr,
	}
	svc := newScopedTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.Deposit(context.Background(), DepositInput{
		Actor: Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}, SavingsAccountID: account.ID,
		Amount: 100_000, TransactionDate: time.Now(),
	})
	if !errors.Is(err, commitErr) {
		t.Fatalf("Deposit() error = %v, want %v", err, commitErr)
	}
	if uow.commitCalls != 1 || uow.rollbackCalls != 1 {
		t.Fatalf("uow calls = commit %d rollback %d; want 1 1", uow.commitCalls, uow.rollbackCalls)
	}
}

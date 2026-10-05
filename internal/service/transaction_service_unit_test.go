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
		SavingsAccounts:         accountRepo,
		Transactions:            transactionRepo,
		Students:                studentRepo,
		StudentClassHistories:   historyRepo,
		TeacherClassAssignments: assignmentRepo,
		AuditLogs:               auditRepo,
	}
}

func TestTransactionServiceDepositAdmin(t *testing.T) {
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
	svc := newTransactionServiceForTest(t, uow.repos, manager)
	actor := Actor{UserID: uuid.New(), Role: domain.RoleAdmin}

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
	svc := newTransactionServiceForTest(t, uow.repos, manager)

	err := WithinTransaction(
		context.Background(),
		manager,
		func(ctx context.Context, _ repository.RepositorySet) error {
			_, err := svc.Deposit(ctx, DepositInput{
				Actor:            Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
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
	svc := newTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.Withdrawal(context.Background(), WithdrawalInput{
		Actor:            Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
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
	svc := newTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	got, err := svc.Withdrawal(context.Background(), WithdrawalInput{
		Actor:            Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
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
		listBySavingsAccountFn: func(context.Context, uuid.UUID, repository.ListOptions) ([]domain.Transaction, error) {
			return []domain.Transaction{transaction}, nil
		},
	}
	svc := newTransactionServiceForTest(t,
		newTransactionTestRepositories(accountRepo, transactionRepo, nil, nil, nil, nil),
		&transactionServiceFakeUnitOfWorkManager{},
	)

	got, err := svc.List(context.Background(), ListTransactionsInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, SavingsAccountID: account.ID,
		Options: ListOptions{Limit: 10, Offset: 20},
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got.Transactions) != 1 || got.Transactions[0].ID != transaction.ID {
		t.Fatalf("List() output = %#v, want one transaction", got.Transactions)
	}
	if transactionRepo.lastListOptions.Limit != 10 || transactionRepo.lastListOptions.Offset != 20 {
		t.Fatalf("List() options = %#v, want limit 10 offset 20", transactionRepo.lastListOptions)
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
	svc := newTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	got, err := svc.Edit(context.Background(), EditTransactionInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, TransactionID: original.ID,
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
	svc := newTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.Edit(context.Background(), EditTransactionInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, TransactionID: original.ID,
		Amount: 50_000, TransactionDate: time.Now(),
	})
	if !errors.Is(err, domain.ErrNegativeBalance) {
		t.Fatalf("Edit() error = %v, want %v", err, domain.ErrNegativeBalance)
	}
	if transactionRepo.updateCalls != 0 || uow.commitCalls != 0 || uow.rollbackCalls != 1 {
		t.Fatalf("calls = update %d commit %d rollback %d; want 0 0 1", transactionRepo.updateCalls, uow.commitCalls, uow.rollbackCalls)
	}
}

func TestTransactionServiceCancel(t *testing.T) {
	account := newTransactionTestAccount(t)
	if err := account.Settle(time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("SavingsAccount.Settle() error = %v", err)
	}

	original := newTransactionTestDeposit(t, account.ID, 100_000)
	current := original
	transactionRepo := &transactionServiceFakeTransactionRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Transaction, error) { return current, nil },
	}
	accountRepo := &transactionServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}
	auditRepo := &transactionServiceFakeAuditRepository{}
	uow := &transactionServiceFakeUnitOfWork{repos: newTransactionTestRepositories(accountRepo, transactionRepo, nil, nil, nil, auditRepo)}
	svc := newTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	got, err := svc.Cancel(context.Background(), CancelTransactionInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, TransactionID: original.ID,
	})
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if got.Transaction.Status != domain.TransactionCancelled {
		t.Fatalf("Cancel() status = %v, want %v", got.Transaction.Status, domain.TransactionCancelled)
	}
	if transactionRepo.updateCalls != 1 || auditRepo.createCalls != 1 || uow.commitCalls != 1 || uow.rollbackCalls != 0 {
		t.Fatalf("calls = update %d audit %d commit %d rollback %d; want 1 1 1 0", transactionRepo.updateCalls, auditRepo.createCalls, uow.commitCalls, uow.rollbackCalls)
	}
	if auditRepo.lastAudit.Action != "CANCEL" || len(auditRepo.lastAudit.BeforeData) == 0 || len(auditRepo.lastAudit.AfterData) == 0 {
		t.Fatalf("audit = %#v, want CANCEL with before/after", auditRepo.lastAudit)
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
	svc := newTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.Deposit(context.Background(), DepositInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, SavingsAccountID: account.ID,
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
	svc := newTransactionServiceForTest(t, repository.RepositorySet{}, &transactionServiceFakeUnitOfWorkManager{beginErr: beginErr})
	_, err := svc.Deposit(context.Background(), DepositInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, SavingsAccountID: uuid.New(), Amount: 1,
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
	svc := newTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.Cancel(context.Background(), CancelTransactionInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, TransactionID: transaction.ID,
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
	svc := newTransactionServiceForTest(t, uow.repos, &transactionServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.Deposit(context.Background(), DepositInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, SavingsAccountID: account.ID,
		Amount: 100_000, TransactionDate: time.Now(),
	})
	if !errors.Is(err, commitErr) {
		t.Fatalf("Deposit() error = %v, want %v", err, commitErr)
	}
	if uow.commitCalls != 1 || uow.rollbackCalls != 1 {
		t.Fatalf("uow calls = commit %d rollback %d; want 1 1", uow.commitCalls, uow.rollbackCalls)
	}
}

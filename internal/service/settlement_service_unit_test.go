package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type settlementServiceFakeSavingsAccountRepository struct {
	repository.SavingsAccountRepository

	getByIDFn          func(context.Context, uuid.UUID) (domain.SavingsAccount, error)
	getByIDForUpdateFn func(context.Context, uuid.UUID) (domain.SavingsAccount, error)
	updateFn           func(context.Context, domain.SavingsAccount) error

	getByIDCalls          int
	getByIDForUpdateCalls int
	updateCalls           int
	lastUpdated           domain.SavingsAccount
}

func (f *settlementServiceFakeSavingsAccountRepository) GetByID(
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

func (f *settlementServiceFakeSavingsAccountRepository) GetByIDForUpdate(
	ctx context.Context,
	id uuid.UUID,
) (domain.SavingsAccount, error) {
	f.getByIDForUpdateCalls++
	if f.getByIDForUpdateFn != nil {
		return f.getByIDForUpdateFn(ctx, id)
	}
	return domain.SavingsAccount{}, repository.ErrNotFound
}

func (f *settlementServiceFakeSavingsAccountRepository) Update(
	ctx context.Context,
	account domain.SavingsAccount,
) error {
	f.updateCalls++
	f.lastUpdated = account
	if f.updateFn != nil {
		return f.updateFn(ctx, account)
	}
	return nil
}

type settlementServiceFakeAcademicYearRepository struct {
	repository.AcademicYearRepository

	getByIDFn func(context.Context, uuid.UUID) (domain.AcademicYear, error)
	getCalls  int
}

func (f *settlementServiceFakeAcademicYearRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.AcademicYear, error) {
	f.getCalls++
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return domain.AcademicYear{}, repository.ErrNotFound
}

func (f *settlementServiceFakeAcademicYearRepository) GetByIDForShare(ctx context.Context, id uuid.UUID) (domain.AcademicYear, error) {
	return f.GetByID(ctx, id)
}

type settlementServiceFakeStudentRepository struct {
	repository.StudentRepository

	getByIDFn func(context.Context, uuid.UUID) (domain.Student, error)
	getCalls  int
}

func (f *settlementServiceFakeStudentRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.Student, error) {
	f.getCalls++
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return domain.Student{}, repository.ErrNotFound
}

type settlementServiceFakeTransactionRepository struct {
	repository.TransactionRepository

	listActiveFn func(context.Context, uuid.UUID) ([]domain.Transaction, error)
	listCalls    int
}

func (f *settlementServiceFakeTransactionRepository) ListActiveBySavingsAccount(
	ctx context.Context,
	id uuid.UUID,
) ([]domain.Transaction, error) {
	f.listCalls++
	if f.listActiveFn != nil {
		return f.listActiveFn(ctx, id)
	}
	return nil, nil
}

type settlementServiceFakeSettlementRepository struct {
	repository.SettlementRepository

	createFn                       func(context.Context, domain.SavingsSettlement) error
	getByIDFn                      func(context.Context, uuid.UUID) (domain.SavingsSettlement, error)
	getCompletedBySavingsAccountFn func(context.Context, uuid.UUID) (domain.SavingsSettlement, error)
	listBySavingsAccountFn         func(context.Context, uuid.UUID, repository.ListOptions) ([]domain.SavingsSettlement, error)
	updateFn                       func(context.Context, domain.SavingsSettlement) error

	createCalls                       int
	getByIDCalls                      int
	getCompletedBySavingsAccountCalls int
	listCalls                         int
	updateCalls                       int
	lastCreated                       domain.SavingsSettlement
	lastUpdated                       domain.SavingsSettlement
	lastListOptions                   repository.ListOptions
}

func (f *settlementServiceFakeSettlementRepository) Create(
	ctx context.Context,
	settlement domain.SavingsSettlement,
) error {
	f.createCalls++
	f.lastCreated = settlement
	if f.createFn != nil {
		return f.createFn(ctx, settlement)
	}
	return nil
}

func (f *settlementServiceFakeSettlementRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.SavingsSettlement, error) {
	f.getByIDCalls++
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return domain.SavingsSettlement{}, repository.ErrNotFound
}

func (f *settlementServiceFakeSettlementRepository) GetCompletedBySavingsAccount(
	ctx context.Context,
	id uuid.UUID,
) (domain.SavingsSettlement, error) {
	f.getCompletedBySavingsAccountCalls++
	if f.getCompletedBySavingsAccountFn != nil {
		return f.getCompletedBySavingsAccountFn(ctx, id)
	}
	return domain.SavingsSettlement{}, repository.ErrNotFound
}

func (f *settlementServiceFakeSettlementRepository) ListBySavingsAccount(
	ctx context.Context,
	id uuid.UUID,
	options repository.ListOptions,
) ([]domain.SavingsSettlement, error) {
	f.listCalls++
	f.lastListOptions = options
	if f.listBySavingsAccountFn != nil {
		return f.listBySavingsAccountFn(ctx, id, options)
	}
	return nil, nil
}

func (f *settlementServiceFakeSettlementRepository) Update(
	ctx context.Context,
	settlement domain.SavingsSettlement,
) error {
	f.updateCalls++
	f.lastUpdated = settlement
	if f.updateFn != nil {
		return f.updateFn(ctx, settlement)
	}
	return nil
}

func (f *settlementServiceFakeSettlementRepository) ListCashMovements(context.Context, uuid.UUID) ([]domain.SettlementCashMovement, error) {
	return nil, nil
}

func (f *settlementServiceFakeSettlementRepository) GetApprovedReopenRequestByAccount(context.Context, uuid.UUID) (domain.SettlementReopenRequest, error) {
	return domain.SettlementReopenRequest{}, repository.ErrNotFound
}
func (f *settlementServiceFakeSettlementRepository) GetActiveReopenRequest(context.Context, uuid.UUID) (domain.SettlementReopenRequest, error) {
	return domain.SettlementReopenRequest{}, repository.ErrNotFound
}
func (f *settlementServiceFakeSettlementRepository) GetLatestReopenRequest(context.Context, uuid.UUID) (domain.SettlementReopenRequest, error) {
	return domain.SettlementReopenRequest{}, repository.ErrNotFound
}

type settlementServiceFakeAuditRepository struct {
	repository.AuditRepository

	createFn    func(context.Context, domain.AuditLog) error
	createCalls int
	audits      []domain.AuditLog
}

func (f *settlementServiceFakeAuditRepository) Create(
	ctx context.Context,
	auditLog domain.AuditLog,
) error {
	f.createCalls++
	f.audits = append(f.audits, auditLog)
	if f.createFn != nil {
		return f.createFn(ctx, auditLog)
	}
	return nil
}

type settlementServiceFakeStudentClassHistoryRepository struct {
	repository.StudentClassHistoryRepository

	listFn func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.StudentClassHistory, error)
}

func (f *settlementServiceFakeStudentClassHistoryRepository) ListByStudentAndAcademicYear(
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

func (f *settlementServiceFakeStudentClassHistoryRepository) GetCurrentByStudentAndAcademicYear(ctx context.Context, studentID, yearID uuid.UUID, _ time.Time) (domain.StudentClassHistory, error) {
	rows, err := f.ListByStudentAndAcademicYear(ctx, studentID, yearID, repository.ListOptions{})
	if err != nil {
		return domain.StudentClassHistory{}, err
	}
	if len(rows) == 0 {
		return domain.StudentClassHistory{}, repository.ErrNotFound
	}
	return rows[0], nil
}

type settlementServiceFakeTeacherAssignmentRepository struct {
	repository.TeacherClassAssignmentRepository

	listByClassFn func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.TeacherClassAssignment, error)
}

func (f *settlementServiceFakeTeacherAssignmentRepository) ListByClassAndAcademicYear(
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

type settlementServiceFakeUnitOfWork struct {
	repos repository.RepositorySet

	commitErr   error
	rollbackErr error

	commitCalls   int
	rollbackCalls int
}

func (f *settlementServiceFakeUnitOfWork) Repositories() repository.RepositorySet {
	return f.repos
}

func (f *settlementServiceFakeUnitOfWork) Commit() error {
	f.commitCalls++
	return f.commitErr
}

func (f *settlementServiceFakeUnitOfWork) Rollback() error {
	f.rollbackCalls++
	return f.rollbackErr
}

type settlementServiceFakeUnitOfWorkManager struct {
	uow repository.UnitOfWork

	beginErr   error
	beginCalls int
}

func (f *settlementServiceFakeUnitOfWorkManager) Begin(context.Context) (repository.UnitOfWork, error) {
	f.beginCalls++
	if f.beginErr != nil {
		return nil, f.beginErr
	}
	return f.uow, nil
}

func newSettlementServiceForTest(
	t *testing.T,
	repos repository.RepositorySet,
	manager repository.UnitOfWorkManager,
) *settlementService {
	t.Helper()

	svc, err := NewSettlementService(Dependencies{
		Repositories: repos,
		UOW:          manager,
	})
	if err != nil {
		t.Fatalf("NewSettlementService() error = %v", err)
	}

	return svc
}

func newSettlementTestAccount(t *testing.T) domain.SavingsAccount {
	t.Helper()

	account, err := domain.NewSavingsAccount(uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}
	return account
}

func newSettlementTestAcademicYear(t *testing.T, id uuid.UUID, status domain.AcademicYearStatus) domain.AcademicYear {
	t.Helper()

	return domain.AcademicYear{
		ID:        id,
		Name:      "2026/2027",
		StartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC),
		Status:    status,
	}
}

func newSettlementTestStudent(id uuid.UUID, status domain.StudentStatus) domain.Student {
	userID := uuid.New()
	return domain.Student{
		ID:     id,
		UserID: &userID,
		Name:   "Ahmad",
		Status: status,
	}
}

func newSettlementTestTransaction(
	t *testing.T,
	accountID uuid.UUID,
	type_ domain.TransactionType,
	amount domain.Money,
) domain.Transaction {
	t.Helper()

	transaction, err := domain.NewTransaction(
		accountID,
		type_,
		amount,
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		uuid.New(),
	)
	if err != nil {
		t.Fatalf("NewTransaction() error = %v", err)
	}
	return transaction
}

func newCompletedSettlement(t *testing.T, accountID, actorID uuid.UUID, amount domain.Money) domain.SavingsSettlement {
	t.Helper()

	settlement, err := domain.NewSavingsSettlement(
		accountID,
		domain.SettlementRegularYearEnd,
		amount,
		actorID,
		time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement() error = %v", err)
	}
	return settlement
}

func newSettlementScopeRepositories(
	history repository.StudentClassHistoryRepository,
	assignment repository.TeacherClassAssignmentRepository,
) repository.RepositorySet {
	return repository.RepositorySet{
		StudentClassHistories:   history,
		TeacherClassAssignments: assignment,
	}
}

func TestSettlementServiceSettleYearEndWaliKelas(t *testing.T) {
	account := newSettlementTestAccount(t)
	academicYear := newSettlementTestAcademicYear(t, account.AcademicYearID, domain.AcademicYearClosing)
	actor := Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}
	transactions := []domain.Transaction{
		newSettlementTestTransaction(t, account.ID, domain.TransactionDeposit, 100000),
		newSettlementTestTransaction(t, account.ID, domain.TransactionWithdrawal, 25000),
	}

	savingsRepo := &settlementServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) {
			return account, nil
		},
	}
	academicRepo := &settlementServiceFakeAcademicYearRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.AcademicYear, error) {
			return academicYear, nil
		},
	}
	transactionRepo := &settlementServiceFakeTransactionRepository{
		listActiveFn: func(context.Context, uuid.UUID) ([]domain.Transaction, error) {
			return transactions, nil
		},
	}
	settlementRepo := &settlementServiceFakeSettlementRepository{}
	auditRepo := &settlementServiceFakeAuditRepository{}
	uow := &settlementServiceFakeUnitOfWork{
		repos: repository.RepositorySet{
			SavingsAccounts: savingsRepo,
			AcademicYears:   academicRepo,
			Transactions:    transactionRepo,
			Settlements:     settlementRepo,
			AuditLogs:       auditRepo,
		},
	}
	svc := newScopedSettlementServiceForTest(
		t,
		uow.repos,
		&settlementServiceFakeUnitOfWorkManager{uow: uow},
	)

	got, err := svc.SettleYearEnd(context.Background(), SettleYearEndInput{
		Actor:            actor,
		SavingsAccountID: account.ID,
		ExpectedBalance:  75000,
	})
	if err != nil {
		t.Fatalf("SettleYearEnd() error = %v", err)
	}
	if got.Settlement == nil || got.Settlement.Type != domain.SettlementRegularYearEnd {
		t.Fatalf("Settlement = %#v, want regular year end settlement", got.Settlement)
	}
	if got.Settlement.Amount != 75000 {
		t.Fatalf("Settlement.Amount = %d, want 75000", got.Settlement.Amount)
	}
	if got.Account == nil || !got.Account.IsSettled() {
		t.Fatalf("Account = %#v, want settled account", got.Account)
	}
	if savingsRepo.getByIDForUpdateCalls != 1 {
		t.Fatalf("GetByIDForUpdate calls = %d, want 1", savingsRepo.getByIDForUpdateCalls)
	}
	if settlementRepo.createCalls != 1 {
		t.Fatalf("Settlements.Create calls = %d, want 1", settlementRepo.createCalls)
	}
	if auditRepo.createCalls != 2 {
		t.Fatalf("AuditLogs.Create calls = %d, want 2", auditRepo.createCalls)
	}
	if uow.commitCalls != 1 || uow.rollbackCalls != 0 {
		t.Fatalf("Commit/Rollback = %d/%d, want 1/0", uow.commitCalls, uow.rollbackCalls)
	}

	if auditRepo.audits[0].Action != "SETTLE" || auditRepo.audits[0].EntityType != domain.AuditEntitySavingsSettlement {
		t.Fatalf("first audit = %#v, want settlement SETTLE audit", auditRepo.audits[0])
	}
	if auditRepo.audits[1].Action != "SETTLE" || auditRepo.audits[1].EntityType != domain.AuditEntitySavingsAccount {
		t.Fatalf("second audit = %#v, want account SETTLE audit", auditRepo.audits[1])
	}
	if len(auditRepo.audits[1].BeforeData) == 0 {
		t.Fatal("account SETTLE audit BeforeData is empty")
	}
}

func TestSettlementServiceSettleYearEndRejectsOpenAcademicYear(t *testing.T) {
	account := newSettlementTestAccount(t)
	academicYear := newSettlementTestAcademicYear(t, account.AcademicYearID, domain.AcademicYearOpen)
	uow := &settlementServiceFakeUnitOfWork{repos: repository.RepositorySet{
		SavingsAccounts: &settlementServiceFakeSavingsAccountRepository{
			getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
		},
		AcademicYears: &settlementServiceFakeAcademicYearRepository{
			getByIDFn: func(context.Context, uuid.UUID) (domain.AcademicYear, error) { return academicYear, nil },
		},
		Transactions: &settlementServiceFakeTransactionRepository{},
		Settlements:  &settlementServiceFakeSettlementRepository{},
		AuditLogs:    &settlementServiceFakeAuditRepository{},
	}}
	svc := newScopedSettlementServiceForTest(t, uow.repos, &settlementServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.SettleYearEnd(context.Background(), SettleYearEndInput{
		Actor:            Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas},
		SavingsAccountID: account.ID,
	})
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("SettleYearEnd() error = %v, want ErrInvalidTransition", err)
	}
	if uow.rollbackCalls != 1 || uow.commitCalls != 0 {
		t.Fatalf("Commit/Rollback = %d/%d, want 0/1", uow.commitCalls, uow.rollbackCalls)
	}
}

func TestSettlementServiceSettleYearEndRejectsStudentActor(t *testing.T) {
	svc := newSettlementServiceForTest(t, repository.RepositorySet{}, &settlementServiceFakeUnitOfWorkManager{})

	_, err := svc.SettleYearEnd(context.Background(), SettleYearEndInput{
		Actor:            Actor{UserID: uuid.New(), Role: domain.RoleSiswa},
		SavingsAccountID: uuid.New(),
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("SettleYearEnd() error = %v, want ErrForbidden", err)
	}
}

func TestSettlementServiceSettleYearEndWaliKelasInScope(t *testing.T) {
	account := newSettlementTestAccount(t)
	academicYear := newSettlementTestAcademicYear(t, account.AcademicYearID, domain.AcademicYearClosing)
	actorID := uuid.New()
	classID := uuid.New()

	repos := newSettlementScopeRepositories(
		&settlementServiceFakeStudentClassHistoryRepository{
			listFn: func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.StudentClassHistory, error) {
				return []domain.StudentClassHistory{{StudentID: account.StudentID, AcademicYearID: account.AcademicYearID, ClassID: classID}}, nil
			},
		},
		&settlementServiceFakeTeacherAssignmentRepository{
			listByClassFn: func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.TeacherClassAssignment, error) {
				return []domain.TeacherClassAssignment{{UserID: actorID, ClassID: classID, AcademicYearID: account.AcademicYearID, Period: domain.DateRange{From: time.Time{}}}}, nil
			},
		},
	)
	repos.SavingsAccounts = &settlementServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}
	repos.AcademicYears = &settlementServiceFakeAcademicYearRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.AcademicYear, error) { return academicYear, nil },
	}
	repos.Transactions = &settlementServiceFakeTransactionRepository{}
	repos.Settlements = &settlementServiceFakeSettlementRepository{}
	repos.AuditLogs = &settlementServiceFakeAuditRepository{}
	uow := &settlementServiceFakeUnitOfWork{repos: repos}
	svc := newSettlementServiceForTest(t, repos, &settlementServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.SettleYearEnd(context.Background(), SettleYearEndInput{
		Actor:            Actor{UserID: actorID, Role: domain.RoleWaliKelas},
		SavingsAccountID: account.ID,
	})
	if err != nil {
		t.Fatalf("SettleYearEnd() error = %v", err)
	}
}

func TestSettlementServiceSettleStudentLeaving(t *testing.T) {
	account := newSettlementTestAccount(t)
	student := newSettlementTestStudent(account.StudentID, domain.StudentLeft)
	transactions := []domain.Transaction{
		newSettlementTestTransaction(t, account.ID, domain.TransactionDeposit, 50000),
	}
	savingsRepo := &settlementServiceFakeSavingsAccountRepository{
		getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}
	studentRepo := &settlementServiceFakeStudentRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Student, error) { return student, nil },
	}
	transactionRepo := &settlementServiceFakeTransactionRepository{
		listActiveFn: func(context.Context, uuid.UUID) ([]domain.Transaction, error) { return transactions, nil },
	}
	settlementRepo := &settlementServiceFakeSettlementRepository{}
	auditRepo := &settlementServiceFakeAuditRepository{}
	uow := &settlementServiceFakeUnitOfWork{repos: repository.RepositorySet{
		SavingsAccounts: savingsRepo,
		AcademicYears: &settlementServiceFakeAcademicYearRepository{getByIDFn: func(context.Context, uuid.UUID) (domain.AcademicYear, error) {
			return newSettlementTestAcademicYear(t, account.AcademicYearID, domain.AcademicYearOpen), nil
		}},
		Students:     studentRepo,
		Transactions: transactionRepo,
		Settlements:  settlementRepo,
		AuditLogs:    auditRepo,
	}}
	svc := newScopedSettlementServiceForTest(t, uow.repos, &settlementServiceFakeUnitOfWorkManager{uow: uow})

	got, err := svc.SettleStudentLeaving(context.Background(), SettleStudentLeavingInput{
		Actor:            Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas},
		SavingsAccountID: account.ID,
	})
	if err != nil {
		t.Fatalf("SettleStudentLeaving() error = %v", err)
	}
	if got.Settlement == nil || got.Settlement.Type != domain.SettlementStudentLeaving {
		t.Fatalf("Settlement = %#v, want student leaving settlement", got.Settlement)
	}
	if got.Settlement.Amount != 50000 {
		t.Fatalf("Settlement.Amount = %d, want 50000", got.Settlement.Amount)
	}
	if got.Student == nil || !got.Student.IsLeft() {
		t.Fatalf("Student = %#v, want left student", got.Student)
	}
	if !got.Account.IsSettled() {
		t.Fatalf("Account status = %v, want SETTLED", got.Account.Status)
	}
	if uow.commitCalls != 1 || uow.rollbackCalls != 0 {
		t.Fatalf("Commit/Rollback = %d/%d, want 1/0", uow.commitCalls, uow.rollbackCalls)
	}
}

func TestSettlementServiceSettleStudentLeavingRejectsActiveStudent(t *testing.T) {
	account := newSettlementTestAccount(t)
	student := newSettlementTestStudent(account.StudentID, domain.StudentActive)
	uow := &settlementServiceFakeUnitOfWork{repos: repository.RepositorySet{
		SavingsAccounts: &settlementServiceFakeSavingsAccountRepository{
			getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
		},
		AcademicYears: &settlementServiceFakeAcademicYearRepository{getByIDFn: func(context.Context, uuid.UUID) (domain.AcademicYear, error) {
			return newSettlementTestAcademicYear(t, account.AcademicYearID, domain.AcademicYearOpen), nil
		}},
		Students: &settlementServiceFakeStudentRepository{
			getByIDFn: func(context.Context, uuid.UUID) (domain.Student, error) { return student, nil },
		},
	}}
	svc := newScopedSettlementServiceForTest(t, uow.repos, &settlementServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.SettleStudentLeaving(context.Background(), SettleStudentLeavingInput{
		Actor:            Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas},
		SavingsAccountID: account.ID,
	})
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("SettleStudentLeaving() error = %v, want ErrInvalidTransition", err)
	}
	if uow.rollbackCalls != 1 {
		t.Fatalf("Rollback calls = %d, want 1", uow.rollbackCalls)
	}
}

func TestSettlementServiceGetStudentSelf(t *testing.T) {
	account := newSettlementTestAccount(t)
	studentUserID := uuid.New()
	student := newSettlementTestStudent(account.StudentID, domain.StudentActive)
	student.UserID = &studentUserID
	settlement := newCompletedSettlement(t, account.ID, uuid.New(), 10000)

	svc := newSettlementServiceForTest(t, repository.RepositorySet{
		Settlements: &settlementServiceFakeSettlementRepository{
			getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsSettlement, error) { return settlement, nil },
		},
		SavingsAccounts: &settlementServiceFakeSavingsAccountRepository{
			getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
		},
		Students: &settlementServiceFakeStudentRepository{
			getByIDFn: func(context.Context, uuid.UUID) (domain.Student, error) { return student, nil },
		},
	}, &settlementServiceFakeUnitOfWorkManager{})

	got, err := svc.Get(context.Background(), GetSettlementInput{
		Actor:        Actor{UserID: studentUserID, Role: domain.RoleSiswa},
		SettlementID: settlement.ID,
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Settlement == nil || got.Settlement.ID != settlement.ID {
		t.Fatalf("Settlement = %#v, want %v", got.Settlement, settlement.ID)
	}
}

func TestSettlementServiceGetStudentOtherUserRejected(t *testing.T) {
	account := newSettlementTestAccount(t)
	studentUserID := uuid.New()
	student := newSettlementTestStudent(account.StudentID, domain.StudentActive)
	student.UserID = &studentUserID
	settlement := newCompletedSettlement(t, account.ID, uuid.New(), 10000)

	svc := newSettlementServiceForTest(t, repository.RepositorySet{
		Settlements: &settlementServiceFakeSettlementRepository{
			getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsSettlement, error) { return settlement, nil },
		},
		SavingsAccounts: &settlementServiceFakeSavingsAccountRepository{
			getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
		},
		Students: &settlementServiceFakeStudentRepository{
			getByIDFn: func(context.Context, uuid.UUID) (domain.Student, error) { return student, nil },
		},
	}, &settlementServiceFakeUnitOfWorkManager{})

	_, err := svc.Get(context.Background(), GetSettlementInput{
		Actor:        Actor{UserID: uuid.New(), Role: domain.RoleSiswa},
		SettlementID: settlement.ID,
	})
	if !errors.Is(err, ErrScopeViolation) {
		t.Fatalf("Get() error = %v, want ErrScopeViolation", err)
	}
}

func TestSettlementServiceListPassesOptions(t *testing.T) {
	account := newSettlementTestAccount(t)
	settlement := newCompletedSettlement(t, account.ID, uuid.New(), 10000)
	settlementRepo := &settlementServiceFakeSettlementRepository{
		listBySavingsAccountFn: func(context.Context, uuid.UUID, repository.ListOptions) ([]domain.SavingsSettlement, error) {
			return []domain.SavingsSettlement{settlement}, nil
		},
	}
	svc := newSettlementServiceForTest(t, repository.RepositorySet{
		SavingsAccounts: &settlementServiceFakeSavingsAccountRepository{
			getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
		},
		Settlements: settlementRepo,
	}, &settlementServiceFakeUnitOfWorkManager{})

	got, err := svc.List(context.Background(), ListSettlementsInput{
		Actor:            Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		SavingsAccountID: account.ID,
		Options:          ListOptions{Limit: 10, Offset: 20},
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got.Settlements) != 1 || got.Settlements[0].ID != settlement.ID {
		t.Fatalf("Settlements = %#v, want one settlement %v", got.Settlements, settlement.ID)
	}
	if settlementRepo.lastListOptions.Limit != 10 || settlementRepo.lastListOptions.Offset != 20 {
		t.Fatalf("ListOptions = %#v, want limit 10 offset 20", settlementRepo.lastListOptions)
	}
}

func TestSettlementServiceMutationAuditErrorRollsBack(t *testing.T) {
	wantErr := errors.New("audit create failed")
	account := newSettlementTestAccount(t)
	academicYear := newSettlementTestAcademicYear(t, account.AcademicYearID, domain.AcademicYearClosing)
	auditRepo := &settlementServiceFakeAuditRepository{
		createFn: func(context.Context, domain.AuditLog) error { return wantErr },
	}
	uow := &settlementServiceFakeUnitOfWork{repos: repository.RepositorySet{
		SavingsAccounts: &settlementServiceFakeSavingsAccountRepository{
			getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
		},
		AcademicYears: &settlementServiceFakeAcademicYearRepository{
			getByIDFn: func(context.Context, uuid.UUID) (domain.AcademicYear, error) { return academicYear, nil },
		},
		Transactions: &settlementServiceFakeTransactionRepository{},
		Settlements:  &settlementServiceFakeSettlementRepository{},
		AuditLogs:    auditRepo,
	}}
	svc := newScopedSettlementServiceForTest(t, uow.repos, &settlementServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.SettleYearEnd(context.Background(), SettleYearEndInput{
		Actor:            Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas},
		SavingsAccountID: account.ID,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("SettleYearEnd() error = %v, want %v", err, wantErr)
	}
	if uow.commitCalls != 0 || uow.rollbackCalls != 1 {
		t.Fatalf("Commit/Rollback = %d/%d, want 0/1", uow.commitCalls, uow.rollbackCalls)
	}
}

func TestSettlementServiceAuditPayloadsAreValidJSON(t *testing.T) {
	account := newSettlementTestAccount(t)
	academicYear := newSettlementTestAcademicYear(t, account.AcademicYearID, domain.AcademicYearClosing)
	auditRepo := &settlementServiceFakeAuditRepository{}
	uow := &settlementServiceFakeUnitOfWork{repos: repository.RepositorySet{
		SavingsAccounts: &settlementServiceFakeSavingsAccountRepository{
			getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
		},
		AcademicYears: &settlementServiceFakeAcademicYearRepository{
			getByIDFn: func(context.Context, uuid.UUID) (domain.AcademicYear, error) { return academicYear, nil },
		},
		Transactions: &settlementServiceFakeTransactionRepository{},
		Settlements:  &settlementServiceFakeSettlementRepository{},
		AuditLogs:    auditRepo,
	}}
	svc := newScopedSettlementServiceForTest(t, uow.repos, &settlementServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.SettleYearEnd(context.Background(), SettleYearEndInput{
		Actor:            Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas},
		SavingsAccountID: account.ID,
	})
	if err != nil {
		t.Fatalf("SettleYearEnd() error = %v", err)
	}
	if len(auditRepo.audits) != 2 {
		t.Fatalf("audit count = %d, want 2", len(auditRepo.audits))
	}
	for i, audit := range auditRepo.audits {
		if !json.Valid(audit.AfterData) {
			t.Fatalf("audit %d AfterData is invalid JSON: %s", i, audit.AfterData)
		}
	}
}

type settlementReadinessRepositoryFake struct {
	repository.SettlementRepository
	rows           []repository.SettlementReadinessAccount
	pending        int
	unreconciled   bool
	activeReopen   bool
	countedTeacher uuid.UUID
}

func (f *settlementReadinessRepositoryFake) ListYearReadiness(context.Context, uuid.UUID) ([]repository.SettlementReadinessAccount, error) {
	return f.rows, nil
}
func (f *settlementReadinessRepositoryFake) CountPendingTransfersForTeacher(_ context.Context, _, teacherID uuid.UUID) (int, error) {
	f.countedTeacher = teacherID
	return f.pending, nil
}
func (f *settlementReadinessRepositoryFake) HasUnreconciledPositiveSettlements(context.Context, uuid.UUID) (bool, error) {
	return f.unreconciled, nil
}
func (f *settlementReadinessRepositoryFake) HasActiveReopenWorkflow(context.Context, uuid.UUID) (bool, error) {
	return f.activeReopen, nil
}

func TestSettlementServiceYearReadinessScopesWaliAndIncludesTransferBlocker(t *testing.T) {
	yearID, waliID, otherWaliID := uuid.New(), uuid.New(), uuid.New()
	classID := uuid.New()
	amount := domain.Money(500)
	rows := []repository.SettlementReadinessAccount{
		{Account: domain.SavingsAccount{AcademicYearID: yearID, Status: domain.SavingsAccountSettled},
			Student: domain.Student{Name: "Siswa cakupan"}, ClassID: &classID, ClassName: "VII A", WaliID: &waliID, WaliName: "Wali A",
			Settlement: &domain.SavingsSettlement{Amount: amount}, CashDelivered: amount},
		{Account: domain.SavingsAccount{AcademicYearID: yearID, Status: domain.SavingsAccountOpen},
			Student: domain.Student{Name: "Siswa wali lain"}, ClassID: &classID, ClassName: "VII A", WaliID: &otherWaliID, WaliName: "Wali B"},
	}
	year := newSettlementTestAcademicYear(t, yearID, domain.AcademicYearClosing)
	readinessRepo := &settlementReadinessRepositoryFake{rows: rows, pending: 1}
	svc := newSettlementServiceForTest(t, repository.RepositorySet{
		AcademicYears: &settlementServiceFakeAcademicYearRepository{getByIDFn: func(context.Context, uuid.UUID) (domain.AcademicYear, error) { return year, nil }},
		Settlements:   readinessRepo,
	}, &settlementServiceFakeUnitOfWorkManager{})
	got, err := svc.YearReadiness(context.Background(), YearSettlementReadinessInput{Actor: Actor{UserID: waliID, Role: domain.RoleWaliKelas}, AcademicYearID: yearID})
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 1 || got.Outstanding != 0 || got.AwaitingHandover != 0 || got.PendingTransfers != 1 || got.Ready {
		t.Fatalf("readiness = %+v; want only wali's row and pending transfer blocker", got)
	}
	if readinessRepo.countedTeacher != waliID {
		t.Fatalf("transfer count teacher = %s, want %s", readinessRepo.countedTeacher, waliID)
	}
}

func TestSettlementServiceYearReadinessAdminSeesUnattributedAndGlobalBlockers(t *testing.T) {
	yearID := uuid.New()
	year := newSettlementTestAcademicYear(t, yearID, domain.AcademicYearClosing)
	readinessRepo := &settlementReadinessRepositoryFake{
		rows: []repository.SettlementReadinessAccount{{
			Account: domain.SavingsAccount{AcademicYearID: yearID, Status: domain.SavingsAccountOpen},
			Student: domain.Student{Name: "Tanpa penempatan"}, AttributionIssue: true,
		}},
		unreconciled: true,
	}
	svc := newSettlementServiceForTest(t, repository.RepositorySet{
		AcademicYears: &settlementServiceFakeAcademicYearRepository{getByIDFn: func(context.Context, uuid.UUID) (domain.AcademicYear, error) { return year, nil }},
		Settlements:   readinessRepo,
	}, &settlementServiceFakeUnitOfWorkManager{})
	got, err := svc.YearReadiness(context.Background(), YearSettlementReadinessInput{Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, AcademicYearID: yearID})
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 1 || got.Unattributed != 1 || got.Outstanding != 1 || got.Ready {
		t.Fatalf("readiness = %+v; want unresolved account and global blockers", got)
	}
}

func TestSettlementServiceYearReadinessRejectsStudent(t *testing.T) {
	svc := newSettlementServiceForTest(t, repository.RepositorySet{}, &settlementServiceFakeUnitOfWorkManager{})
	_, err := svc.YearReadiness(context.Background(), YearSettlementReadinessInput{Actor: Actor{UserID: uuid.New(), Role: domain.RoleSiswa}, AcademicYearID: uuid.New()})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("YearReadiness() error = %v, want forbidden", err)
	}
}

func TestSettlementServiceRejectsStalePreviewAndRollsBack(t *testing.T) {
	account := newSettlementTestAccount(t)
	year := newSettlementTestAcademicYear(t, account.AcademicYearID, domain.AcademicYearClosing)
	tx := newSettlementTestTransaction(t, account.ID, domain.TransactionDeposit, 100)
	accountRepo := &settlementServiceFakeSavingsAccountRepository{}
	accountRepo.getByIDForUpdateFn = func(context.Context, uuid.UUID) (domain.SavingsAccount, error) {
		if accountRepo.updateCalls > 0 {
			return accountRepo.lastUpdated, nil
		}
		return account, nil
	}
	settlementRepo := &settlementServiceFakeSettlementRepository{}
	uow := &settlementServiceFakeUnitOfWork{repos: repository.RepositorySet{
		SavingsAccounts: accountRepo,
		AcademicYears:   &settlementServiceFakeAcademicYearRepository{getByIDFn: func(context.Context, uuid.UUID) (domain.AcademicYear, error) { return year, nil }},
		Transactions:    &settlementServiceFakeTransactionRepository{listActiveFn: func(context.Context, uuid.UUID) ([]domain.Transaction, error) { return []domain.Transaction{tx}, nil }},
		Settlements:     settlementRepo, AuditLogs: &settlementServiceFakeAuditRepository{},
	}}
	svc := newScopedSettlementServiceForTest(t, uow.repos, &settlementServiceFakeUnitOfWorkManager{uow: uow})
	_, err := svc.SettleYearEnd(context.Background(), SettleYearEndInput{Actor: Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}, SavingsAccountID: account.ID, ExpectedBalance: 99})
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("SettleYearEnd() error = %v, want stale-preview conflict", err)
	}
	if settlementRepo.createCalls != 0 || accountRepo.updateCalls != 0 || uow.rollbackCalls != 1 || uow.commitCalls != 0 {
		t.Fatalf("settlement=%d account-update=%d commit/rollback=%d/%d; stale preview must not mutate", settlementRepo.createCalls, accountRepo.updateCalls, uow.commitCalls, uow.rollbackCalls)
	}
}

func TestSettlementServiceAllowsZeroBalanceAndRejectsDuplicateSettlement(t *testing.T) {
	account := newSettlementTestAccount(t)
	year := newSettlementTestAcademicYear(t, account.AcademicYearID, domain.AcademicYearClosing)
	accountRepo := &settlementServiceFakeSavingsAccountRepository{}
	accountRepo.getByIDForUpdateFn = func(context.Context, uuid.UUID) (domain.SavingsAccount, error) {
		if accountRepo.updateCalls > 0 {
			return accountRepo.lastUpdated, nil
		}
		return account, nil
	}
	settlementRepo := &settlementServiceFakeSettlementRepository{}
	uow := &settlementServiceFakeUnitOfWork{repos: repository.RepositorySet{
		SavingsAccounts: accountRepo,
		AcademicYears:   &settlementServiceFakeAcademicYearRepository{getByIDFn: func(context.Context, uuid.UUID) (domain.AcademicYear, error) { return year, nil }},
		Transactions:    &settlementServiceFakeTransactionRepository{}, Settlements: settlementRepo, AuditLogs: &settlementServiceFakeAuditRepository{},
	}}
	svc := newScopedSettlementServiceForTest(t, uow.repos, &settlementServiceFakeUnitOfWorkManager{uow: uow})
	got, err := svc.SettleYearEnd(context.Background(), SettleYearEndInput{Actor: Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}, SavingsAccountID: account.ID, ExpectedBalance: 0})
	if err != nil {
		t.Fatalf("zero-balance settlement error = %v", err)
	}
	if got.Settlement == nil || got.Settlement.Amount != 0 || got.Account == nil || !got.Account.IsSettled() {
		t.Fatalf("zero-balance result = %+v, want settled zero-value account", got)
	}
	_, err = svc.SettleYearEnd(context.Background(), SettleYearEndInput{Actor: Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}, SavingsAccountID: account.ID, ExpectedBalance: 0})
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("duplicate settlement error = %v, want invalid transition", err)
	}
}

type settlementListTotalsFake struct {
	repository.ListTotalsRepository
	transfers int
}

func (f settlementListTotalsFake) Transfers(context.Context, uuid.UUID, uuid.UUID, bool) (int, error) {
	return f.transfers, nil
}

type settlementTransferMarker struct {
	repository.ClassTransferRequestRepository
}

func TestSettlementServiceAdminReadinessIncludesPendingTransfers(t *testing.T) {
	yearID := uuid.New()
	year := newSettlementTestAcademicYear(t, yearID, domain.AcademicYearClosing)
	svc := newSettlementServiceForTest(t, repository.RepositorySet{
		AcademicYears:         &settlementServiceFakeAcademicYearRepository{getByIDFn: func(context.Context, uuid.UUID) (domain.AcademicYear, error) { return year, nil }},
		Settlements:           &settlementReadinessRepositoryFake{},
		ClassTransferRequests: &settlementTransferMarker{},
		ListTotals:            settlementListTotalsFake{transfers: 2},
	}, &settlementServiceFakeUnitOfWorkManager{})
	got, err := svc.YearReadiness(context.Background(), YearSettlementReadinessInput{Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, AcademicYearID: yearID})
	if err != nil {
		t.Fatal(err)
	}
	if got.PendingTransfers != 2 || got.Ready {
		t.Fatalf("readiness = %+v, want two pending transfers and blocked year", got)
	}
}

func TestSettlementServiceAdminCannotSettleOrExecuteReopen(t *testing.T) {
	admin := Actor{UserID: uuid.New(), Role: domain.RoleAdmin}
	svc := newSettlementServiceForTest(t, repository.RepositorySet{}, &settlementServiceFakeUnitOfWorkManager{})
	if _, err := svc.SettleYearEnd(context.Background(), SettleYearEndInput{Actor: admin, SavingsAccountID: uuid.New()}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Admin SettleYearEnd() error = %v, want forbidden", err)
	}
	if _, err := svc.Reopen(context.Background(), ReopenSettlementInput{Actor: admin, SettlementID: uuid.New()}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Admin Reopen() error = %v, want forbidden", err)
	}
}

func TestSettlementServiceSettleYearEndRejectsWaliOutsideHistoricalScope(t *testing.T) {
	account := newSettlementTestAccount(t)
	year := newSettlementTestAcademicYear(t, account.AcademicYearID, domain.AcademicYearClosing)
	actorID, historicalWaliID, classID := uuid.New(), uuid.New(), uuid.New()
	repos := newSettlementScopeRepositories(
		&settlementServiceFakeStudentClassHistoryRepository{listFn: func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.StudentClassHistory, error) {
			return []domain.StudentClassHistory{{StudentID: account.StudentID, AcademicYearID: account.AcademicYearID, ClassID: classID}}, nil
		}},
		&settlementServiceFakeTeacherAssignmentRepository{listByClassFn: func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.TeacherClassAssignment, error) {
			return []domain.TeacherClassAssignment{{UserID: historicalWaliID, ClassID: classID, AcademicYearID: account.AcademicYearID, Period: domain.DateRange{From: time.Time{}}}}, nil
		}},
	)
	repos.SavingsAccounts = &settlementServiceFakeSavingsAccountRepository{getByIDForUpdateFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil }}
	repos.AcademicYears = &settlementServiceFakeAcademicYearRepository{getByIDFn: func(context.Context, uuid.UUID) (domain.AcademicYear, error) { return year, nil }}
	repos.Transactions = &settlementServiceFakeTransactionRepository{}
	repos.Settlements = &settlementServiceFakeSettlementRepository{}
	uow := &settlementServiceFakeUnitOfWork{repos: repos}
	svc := newSettlementServiceForTest(t, repos, &settlementServiceFakeUnitOfWorkManager{uow: uow})
	_, err := svc.SettleYearEnd(context.Background(), SettleYearEndInput{Actor: Actor{UserID: actorID, Role: domain.RoleWaliKelas}, SavingsAccountID: account.ID, ExpectedBalance: 0})
	if !errors.Is(err, ErrScopeViolation) {
		t.Fatalf("SettleYearEnd() error = %v, want historical-scope violation", err)
	}
	if uow.commitCalls != 0 || uow.rollbackCalls != 1 {
		t.Fatalf("commit/rollback = %d/%d, want 0/1", uow.commitCalls, uow.rollbackCalls)
	}
}

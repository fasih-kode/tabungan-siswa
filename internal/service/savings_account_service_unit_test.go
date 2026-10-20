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

type savingsAccountServiceFakeSavingsAccountRepository struct {
	repository.SavingsAccountRepository

	createFn                      func(context.Context, domain.SavingsAccount) error
	getByIDFn                     func(context.Context, uuid.UUID) (domain.SavingsAccount, error)
	getByStudentAndAcademicYearFn func(context.Context, uuid.UUID, uuid.UUID) (domain.SavingsAccount, error)
	updateFn                      func(context.Context, domain.SavingsAccount) error
	hasOpenByAcademicYearFn       func(context.Context, uuid.UUID) (bool, error)

	createCalls int
}

func (f *savingsAccountServiceFakeSavingsAccountRepository) Create(
	ctx context.Context,
	account domain.SavingsAccount,
) error {
	f.createCalls++
	if f.createFn != nil {
		return f.createFn(ctx, account)
	}
	return nil
}

func (f *savingsAccountServiceFakeSavingsAccountRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.SavingsAccount, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return domain.SavingsAccount{}, repository.ErrNotFound
}

func (f *savingsAccountServiceFakeSavingsAccountRepository) GetByStudentAndAcademicYear(
	ctx context.Context,
	studentID uuid.UUID,
	academicYearID uuid.UUID,
) (domain.SavingsAccount, error) {
	if f.getByStudentAndAcademicYearFn != nil {
		return f.getByStudentAndAcademicYearFn(ctx, studentID, academicYearID)
	}
	return domain.SavingsAccount{}, repository.ErrNotFound
}

func (f *savingsAccountServiceFakeSavingsAccountRepository) HasOpenByAcademicYear(
	ctx context.Context,
	academicYearID uuid.UUID,
) (bool, error) {
	if f.hasOpenByAcademicYearFn != nil {
		return f.hasOpenByAcademicYearFn(ctx, academicYearID)
	}
	return false, nil
}

func (f *savingsAccountServiceFakeSavingsAccountRepository) Update(
	ctx context.Context,
	account domain.SavingsAccount,
) error {
	if f.updateFn != nil {
		return f.updateFn(ctx, account)
	}
	return nil
}

type savingsAccountServiceFakeStudentRepository struct {
	repository.StudentRepository

	getByIDFn func(context.Context, uuid.UUID) (domain.Student, error)
}

func (f *savingsAccountServiceFakeStudentRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (domain.Student, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return domain.Student{}, repository.ErrNotFound
}

type savingsAccountServiceFakeStudentClassHistoryRepository struct {
	repository.StudentClassHistoryRepository

	listFn func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.StudentClassHistory, error)
}

func (f *savingsAccountServiceFakeStudentClassHistoryRepository) ListByStudentAndAcademicYear(
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

func (f *savingsAccountServiceFakeStudentClassHistoryRepository) GetCurrentByStudentAndAcademicYear(ctx context.Context, studentID, yearID uuid.UUID, _ time.Time) (domain.StudentClassHistory, error) {
	rows, err := f.ListByStudentAndAcademicYear(ctx, studentID, yearID, repository.ListOptions{})
	if err != nil {
		return domain.StudentClassHistory{}, err
	}
	if len(rows) == 0 {
		return domain.StudentClassHistory{}, repository.ErrNotFound
	}
	return rows[0], nil
}

type savingsAccountServiceFakeTeacherAssignmentRepository struct {
	repository.TeacherClassAssignmentRepository

	listByClassFn func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.TeacherClassAssignment, error)
}

func (f *savingsAccountServiceFakeTeacherAssignmentRepository) ListByClassAndAcademicYear(
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

type savingsAccountServiceFakeTransactionRepository struct {
	repository.TransactionRepository

	listActiveFn func(context.Context, uuid.UUID) ([]domain.Transaction, error)
}

func (f *savingsAccountServiceFakeTransactionRepository) ListActiveBySavingsAccount(
	ctx context.Context,
	savingsAccountID uuid.UUID,
) ([]domain.Transaction, error) {
	if f.listActiveFn != nil {
		return f.listActiveFn(ctx, savingsAccountID)
	}
	return nil, nil
}

func TestSavingsAccountServicePassbookSortsAndCalculatesRunningBalance(t *testing.T) {
	account := newTestSavingsAccount(t)
	day := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	firstID, secondID := uuid.New(), uuid.New()
	transactions := []domain.Transaction{
		{ID: uuid.New(), SavingsAccountID: account.ID, Type: domain.TransactionWithdrawal, Amount: 25, TransactionDate: day.Add(12 * time.Hour), CreatedAt: day.Add(12 * time.Hour), Status: domain.TransactionActive},
		{ID: secondID, SavingsAccountID: account.ID, Type: domain.TransactionDeposit, Amount: 40, TransactionDate: day, CreatedAt: day.Add(time.Hour), Status: domain.TransactionActive},
		{ID: firstID, SavingsAccountID: account.ID, Type: domain.TransactionDeposit, Amount: 50, TransactionDate: day, CreatedAt: day, Status: domain.TransactionActive},
	}
	repos := repository.RepositorySet{
		SavingsAccounts: &savingsAccountServiceFakeSavingsAccountRepository{getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil }},
		Transactions: &savingsAccountServiceFakeTransactionRepository{listActiveFn: func(_ context.Context, id uuid.UUID) ([]domain.Transaction, error) {
			if id != account.ID {
				t.Fatalf("account ID=%s, want %s", id, account.ID)
			}
			return transactions, nil
		}},
	}
	svc := newSavingsAccountServiceForTest(t, repos, &savingsAccountServiceFakeUnitOfWorkManager{})
	out, err := svc.GetPassbook(context.Background(), GetSavingsAccountPassbookInput{Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, SavingsAccountID: account.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Entries) != 3 || out.Balance != 65 {
		t.Fatalf("entries=%d balance=%d, want 3/65", len(out.Entries), out.Balance)
	}
	if out.Entries[0].Transaction.ID != firstID || out.Entries[0].RunningBalance != 50 || out.Entries[1].Transaction.ID != secondID || out.Entries[1].RunningBalance != 90 || out.Entries[2].RunningBalance != 65 {
		t.Fatalf("passbook order/balances = %+v", out.Entries)
	}
}

func TestSavingsAccountServicePassbookRejectsOtherStudent(t *testing.T) {
	account := newTestSavingsAccount(t)
	ownerID := uuid.New()
	repos := repository.RepositorySet{
		SavingsAccounts: &savingsAccountServiceFakeSavingsAccountRepository{getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil }},
		Students: &savingsAccountServiceFakeStudentRepository{getByIDFn: func(context.Context, uuid.UUID) (domain.Student, error) {
			return domain.Student{ID: account.StudentID, UserID: &ownerID}, nil
		}},
	}
	svc := newSavingsAccountServiceForTest(t, repos, &savingsAccountServiceFakeUnitOfWorkManager{})
	_, err := svc.GetPassbook(context.Background(), GetSavingsAccountPassbookInput{Actor: Actor{UserID: uuid.New(), Role: domain.RoleSiswa}, SavingsAccountID: account.ID})
	if !errors.Is(err, ErrScopeViolation) {
		t.Fatalf("GetPassbook() error=%v, want %v", err, ErrScopeViolation)
	}
}

type savingsAccountServiceFakeAuditRepository struct {
	repository.AuditRepository

	createFn func(context.Context, domain.AuditLog) error

	createCalls int
	lastAudit   domain.AuditLog
}

func (f *savingsAccountServiceFakeAuditRepository) Create(
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

type savingsAccountServiceFakeUnitOfWork struct {
	repos repository.RepositorySet

	commitErr   error
	rollbackErr error

	commitCount   int
	rollbackCount int
}

func (f *savingsAccountServiceFakeUnitOfWork) Repositories() repository.RepositorySet {
	return f.repos
}

func (f *savingsAccountServiceFakeUnitOfWork) Commit() error {
	f.commitCount++
	return f.commitErr
}

func (f *savingsAccountServiceFakeUnitOfWork) Rollback() error {
	f.rollbackCount++
	return f.rollbackErr
}

type savingsAccountServiceFakeUnitOfWorkManager struct {
	uow repository.UnitOfWork

	beginErr   error
	beginCount int
}

func (f *savingsAccountServiceFakeUnitOfWorkManager) Begin(context.Context) (repository.UnitOfWork, error) {
	f.beginCount++
	if f.beginErr != nil {
		return nil, f.beginErr
	}
	return f.uow, nil
}

func newSavingsAccountServiceForTest(
	t *testing.T,
	repos repository.RepositorySet,
	uowManager repository.UnitOfWorkManager,
) *savingsAccountService {
	t.Helper()

	svc, err := NewSavingsAccountService(Dependencies{
		Repositories: repos,
		UOW:          uowManager,
	})
	if err != nil {
		t.Fatalf("NewSavingsAccountService() error = %v", err)
	}

	return svc
}

func newTestSavingsAccount(t *testing.T) domain.SavingsAccount {
	t.Helper()

	account, err := domain.NewSavingsAccount(uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}
	return account
}

func newTestStudent(t *testing.T, id, userID uuid.UUID) domain.Student {
	t.Helper()

	return domain.Student{
		ID:     id,
		UserID: &userID,
		Name:   "Ahmad",
		Status: domain.StudentActive,
	}
}

func newTestScopeRepositories(
	historyRepo repository.StudentClassHistoryRepository,
	assignmentRepo repository.TeacherClassAssignmentRepository,
) repository.RepositorySet {
	return repository.RepositorySet{
		StudentClassHistories:   historyRepo,
		TeacherClassAssignments: assignmentRepo,
	}
}

func TestSavingsAccountServiceCreateWaliKelas(t *testing.T) {
	studentID := uuid.New()
	academicYearID := uuid.New()
	actorID := financialTestTeacherID

	savingsRepo := &savingsAccountServiceFakeSavingsAccountRepository{}
	auditRepo := &savingsAccountServiceFakeAuditRepository{}
	uow := &savingsAccountServiceFakeUnitOfWork{
		repos: repository.RepositorySet{
			SavingsAccounts: savingsRepo,
			AuditLogs:       auditRepo,
		},
	}
	manager := &savingsAccountServiceFakeUnitOfWorkManager{uow: uow}
	svc := newScopedSavingsAccountServiceForTest(
		t,
		uow.repos,
		manager,
	)

	got, err := svc.Create(context.Background(), CreateSavingsAccountInput{
		Actor: Actor{
			UserID: actorID,
			Role:   domain.RoleWaliKelas,
		},
		StudentID:      studentID,
		AcademicYearID: academicYearID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if got.Account == nil {
		t.Fatal("Create() Account = nil")
	}
	if got.Account.StudentID != studentID {
		t.Fatalf("StudentID = %v, want %v", got.Account.StudentID, studentID)
	}
	if got.Account.AcademicYearID != academicYearID {
		t.Fatalf("AcademicYearID = %v, want %v", got.Account.AcademicYearID, academicYearID)
	}
	if !got.Account.IsOpen() {
		t.Fatal("created account is not open")
	}
	if savingsRepo.createCalls != 1 {
		t.Fatalf("SavingsAccounts.Create calls = %d, want 1", savingsRepo.createCalls)
	}
	if auditRepo.createCalls != 1 {
		t.Fatalf("AuditLogs.Create calls = %d, want 1", auditRepo.createCalls)
	}
	if auditRepo.lastAudit.Action != "CREATE" {
		t.Fatalf("audit Action = %q, want CREATE", auditRepo.lastAudit.Action)
	}
	if auditRepo.lastAudit.EntityType != domain.AuditEntitySavingsAccount {
		t.Fatalf("audit EntityType = %q, want %q", auditRepo.lastAudit.EntityType, domain.AuditEntitySavingsAccount)
	}
	if auditRepo.lastAudit.EntityID != got.Account.ID {
		t.Fatalf("audit EntityID = %v, want %v", auditRepo.lastAudit.EntityID, got.Account.ID)
	}
	if auditRepo.lastAudit.ActorUserID == nil || *auditRepo.lastAudit.ActorUserID != actorID {
		t.Fatalf("audit ActorUserID = %v, want %v", auditRepo.lastAudit.ActorUserID, actorID)
	}
	if len(auditRepo.lastAudit.AfterData) == 0 {
		t.Fatal("audit AfterData is empty")
	}
	if manager.beginCount != 1 {
		t.Fatalf("Begin calls = %d, want 1", manager.beginCount)
	}
	if uow.commitCount != 1 {
		t.Fatalf("Commit calls = %d, want 1", uow.commitCount)
	}
	if uow.rollbackCount != 0 {
		t.Fatalf("Rollback calls = %d, want 0", uow.rollbackCount)
	}
}

func TestSavingsAccountServiceCreateRejectsLeftStudent(t *testing.T) {
	studentID := uuid.New()
	academicYearID := uuid.New()
	student := domain.Student{ID: studentID, Name: "LEFT", Status: domain.StudentLeft}
	studentRepo := &savingsAccountServiceFakeStudentRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Student, error) { return student, nil },
	}
	savingsRepo := &savingsAccountServiceFakeSavingsAccountRepository{}
	uow := &savingsAccountServiceFakeUnitOfWork{repos: repository.RepositorySet{SavingsAccounts: savingsRepo, Students: studentRepo}}
	svc := newScopedSavingsAccountServiceForTest(t, uow.repos, &savingsAccountServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.Create(context.Background(), CreateSavingsAccountInput{
		Actor: Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}, StudentID: studentID, AcademicYearID: academicYearID,
	})
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("Create(LEFT student) error = %v, want %v", err, domain.ErrInvalidTransition)
	}
	if savingsRepo.createCalls != 0 || uow.commitCount != 0 {
		t.Fatalf("create calls = %d, commit calls = %d; want 0/0", savingsRepo.createCalls, uow.commitCount)
	}
}

func TestSavingsAccountServiceCreateRejectsClosedAcademicYear(t *testing.T) {
	studentID := uuid.New()
	academicYearID := uuid.New()
	student := domain.Student{ID: studentID, Name: "ACTIVE", Status: domain.StudentActive}
	studentRepo := &savingsAccountServiceFakeStudentRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.Student, error) { return student, nil },
	}
	year := domain.AcademicYear{ID: academicYearID, Status: domain.AcademicYearClosed}
	yearRepo := &classAssignmentFakeAcademicYearRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.AcademicYear, error) { return year, nil },
	}
	savingsRepo := &savingsAccountServiceFakeSavingsAccountRepository{}
	uow := &savingsAccountServiceFakeUnitOfWork{repos: repository.RepositorySet{SavingsAccounts: savingsRepo, Students: studentRepo, AcademicYears: yearRepo}}
	svc := newScopedSavingsAccountServiceForTest(t, uow.repos, &savingsAccountServiceFakeUnitOfWorkManager{uow: uow})

	_, err := svc.Create(context.Background(), CreateSavingsAccountInput{
		Actor: Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas}, StudentID: studentID, AcademicYearID: academicYearID,
	})
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("Create(CLOSED year) error = %v, want %v", err, domain.ErrInvalidTransition)
	}
	if savingsRepo.createCalls != 0 || uow.commitCount != 0 {
		t.Fatalf("create calls = %d, commit calls = %d; want 0/0", savingsRepo.createCalls, uow.commitCount)
	}
}

func TestSavingsAccountServiceCreateWaliKelasInScope(t *testing.T) {
	studentID := uuid.New()
	academicYearID := uuid.New()
	classID := uuid.New()
	actorID := uuid.New()

	historyRepo := &savingsAccountServiceFakeStudentClassHistoryRepository{
		listFn: func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.StudentClassHistory, error) {
			return []domain.StudentClassHistory{{
				StudentID:      studentID,
				AcademicYearID: academicYearID,
				ClassID:        classID,
			}}, nil
		},
	}
	assignmentRepo := &savingsAccountServiceFakeTeacherAssignmentRepository{
		listByClassFn: func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.TeacherClassAssignment, error) {
			return []domain.TeacherClassAssignment{{
				UserID:         actorID,
				ClassID:        classID,
				AcademicYearID: academicYearID,
			}}, nil
		},
	}
	savingsRepo := &savingsAccountServiceFakeSavingsAccountRepository{}
	auditRepo := &savingsAccountServiceFakeAuditRepository{}
	uow := &savingsAccountServiceFakeUnitOfWork{
		repos: repository.RepositorySet{
			SavingsAccounts:         savingsRepo,
			AuditLogs:               auditRepo,
			StudentClassHistories:   historyRepo,
			TeacherClassAssignments: assignmentRepo,
		},
	}

	svc := newSavingsAccountServiceForTest(
		t,
		uow.repos,
		&savingsAccountServiceFakeUnitOfWorkManager{uow: uow},
	)

	got, err := svc.Create(context.Background(), CreateSavingsAccountInput{
		Actor:          Actor{UserID: actorID, Role: domain.RoleWaliKelas},
		StudentID:      studentID,
		AcademicYearID: academicYearID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got.Account == nil {
		t.Fatal("Create() Account = nil")
	}
}

func TestSavingsAccountServiceCreateRejectsForbiddenRole(t *testing.T) {
	svc := newSavingsAccountServiceForTest(
		t,
		repository.RepositorySet{},
		&savingsAccountServiceFakeUnitOfWorkManager{},
	)

	_, err := svc.Create(context.Background(), CreateSavingsAccountInput{
		Actor:          Actor{UserID: uuid.New(), Role: domain.RoleSiswa},
		StudentID:      uuid.New(),
		AcademicYearID: uuid.New(),
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Create() error = %v, want ErrForbidden", err)
	}
}

func TestSavingsAccountServiceCreateRejectsInvalidIDs(t *testing.T) {
	svc := newScopedSavingsAccountServiceForTest(
		t,
		repository.RepositorySet{},
		&savingsAccountServiceFakeUnitOfWorkManager{},
	)

	tests := []struct {
		name           string
		studentID      uuid.UUID
		academicYearID uuid.UUID
	}{
		{"nil student", uuid.Nil, uuid.New()},
		{"nil academic year", uuid.New(), uuid.Nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), CreateSavingsAccountInput{
				Actor:          Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas},
				StudentID:      tt.studentID,
				AcademicYearID: tt.academicYearID,
			})
			if !errors.Is(err, ErrScopeViolation) {
				t.Fatalf("Create() error = %v, want ErrScopeViolation", err)
			}
		})
	}
}

func TestSavingsAccountServiceCreateWaliKelasOutOfScope(t *testing.T) {
	studentID := uuid.New()
	academicYearID := uuid.New()
	classID := uuid.New()

	historyRepo := &savingsAccountServiceFakeStudentClassHistoryRepository{
		listFn: func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.StudentClassHistory, error) {
			return []domain.StudentClassHistory{{
				StudentID:      studentID,
				AcademicYearID: academicYearID,
				ClassID:        classID,
			}}, nil
		},
	}
	assignmentRepo := &savingsAccountServiceFakeTeacherAssignmentRepository{
		listByClassFn: func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.TeacherClassAssignment, error) {
			return nil, nil
		},
	}
	uowManager := &savingsAccountServiceFakeUnitOfWorkManager{}
	svc := newSavingsAccountServiceForTest(
		t,
		newTestScopeRepositories(historyRepo, assignmentRepo),
		uowManager,
	)

	_, err := svc.Create(context.Background(), CreateSavingsAccountInput{
		Actor:          Actor{UserID: uuid.New(), Role: domain.RoleWaliKelas},
		StudentID:      studentID,
		AcademicYearID: academicYearID,
	})
	if !errors.Is(err, ErrScopeViolation) {
		t.Fatalf("Create() error = %v, want ErrScopeViolation", err)
	}
	if uowManager.beginCount != 0 {
		t.Fatalf("Begin calls = %d, want 0", uowManager.beginCount)
	}
}

func TestSavingsAccountServiceCreatePropagatesRepositoryErrorAndRollsBack(t *testing.T) {
	wantErr := errors.New("create savings account failed")
	savingsRepo := &savingsAccountServiceFakeSavingsAccountRepository{
		createFn: func(context.Context, domain.SavingsAccount) error {
			return wantErr
		},
	}
	auditRepo := &savingsAccountServiceFakeAuditRepository{}
	uow := &savingsAccountServiceFakeUnitOfWork{
		repos: repository.RepositorySet{
			SavingsAccounts: savingsRepo,
			AuditLogs:       auditRepo,
		},
	}
	manager := &savingsAccountServiceFakeUnitOfWorkManager{uow: uow}
	svc := newScopedSavingsAccountServiceForTest(t, uow.repos, manager)

	_, err := svc.Create(context.Background(), CreateSavingsAccountInput{
		Actor:          Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas},
		StudentID:      uuid.New(),
		AcademicYearID: uuid.New(),
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want %v", err, wantErr)
	}
	if uow.rollbackCount != 1 {
		t.Fatalf("Rollback calls = %d, want 1", uow.rollbackCount)
	}
	if uow.commitCount != 0 {
		t.Fatalf("Commit calls = %d, want 0", uow.commitCount)
	}
	if auditRepo.createCalls != 0 {
		t.Fatalf("AuditLogs.Create calls = %d, want 0", auditRepo.createCalls)
	}
}

func TestSavingsAccountServiceCreateAuditErrorRollsBack(t *testing.T) {
	wantErr := errors.New("audit create failed")
	auditRepo := &savingsAccountServiceFakeAuditRepository{
		createFn: func(context.Context, domain.AuditLog) error {
			return wantErr
		},
	}
	uow := &savingsAccountServiceFakeUnitOfWork{
		repos: repository.RepositorySet{
			SavingsAccounts: &savingsAccountServiceFakeSavingsAccountRepository{},
			AuditLogs:       auditRepo,
		},
	}
	svc := newScopedSavingsAccountServiceForTest(
		t,
		uow.repos,
		&savingsAccountServiceFakeUnitOfWorkManager{uow: uow},
	)

	_, err := svc.Create(context.Background(), CreateSavingsAccountInput{
		Actor:          Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas},
		StudentID:      uuid.New(),
		AcademicYearID: uuid.New(),
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want %v", err, wantErr)
	}
	if uow.rollbackCount != 1 {
		t.Fatalf("Rollback calls = %d, want 1", uow.rollbackCount)
	}
	if uow.commitCount != 0 {
		t.Fatalf("Commit calls = %d, want 0", uow.commitCount)
	}
}

func TestSavingsAccountServiceCreateCommitErrorRollsBack(t *testing.T) {
	wantErr := errors.New("commit failed")
	uow := &savingsAccountServiceFakeUnitOfWork{
		repos: repository.RepositorySet{
			SavingsAccounts: &savingsAccountServiceFakeSavingsAccountRepository{},
			AuditLogs:       &savingsAccountServiceFakeAuditRepository{},
		},
		commitErr: wantErr,
	}
	svc := newScopedSavingsAccountServiceForTest(
		t,
		uow.repos,
		&savingsAccountServiceFakeUnitOfWorkManager{uow: uow},
	)

	_, err := svc.Create(context.Background(), CreateSavingsAccountInput{
		Actor:          Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas},
		StudentID:      uuid.New(),
		AcademicYearID: uuid.New(),
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want %v", err, wantErr)
	}
	if uow.commitCount != 1 {
		t.Fatalf("Commit calls = %d, want 1", uow.commitCount)
	}
	if uow.rollbackCount != 1 {
		t.Fatalf("Rollback calls = %d, want 1", uow.rollbackCount)
	}
}

func TestSavingsAccountServiceCreateBeginError(t *testing.T) {
	wantErr := errors.New("begin failed")
	manager := &savingsAccountServiceFakeUnitOfWorkManager{beginErr: wantErr}
	svc := newScopedSavingsAccountServiceForTest(t, repository.RepositorySet{}, manager)

	_, err := svc.Create(context.Background(), CreateSavingsAccountInput{
		Actor:          Actor{UserID: financialTestTeacherID, Role: domain.RoleWaliKelas},
		StudentID:      uuid.New(),
		AcademicYearID: uuid.New(),
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want %v", err, wantErr)
	}
	if manager.beginCount != 1 {
		t.Fatalf("Begin calls = %d, want 1", manager.beginCount)
	}
}

func TestSavingsAccountServiceGetAdmin(t *testing.T) {
	account := newTestSavingsAccount(t)
	repo := &savingsAccountServiceFakeSavingsAccountRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) {
			return account, nil
		},
	}
	svc := newSavingsAccountServiceForTest(
		t,
		repository.RepositorySet{SavingsAccounts: repo},
		&savingsAccountServiceFakeUnitOfWorkManager{},
	)

	got, err := svc.Get(context.Background(), GetSavingsAccountInput{
		Actor:            Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		SavingsAccountID: account.ID,
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Account == nil || got.Account.ID != account.ID {
		t.Fatalf("Get() Account = %#v, want account %v", got.Account, account.ID)
	}
}

func TestSavingsAccountServiceGetStudentSelf(t *testing.T) {
	account := newTestSavingsAccount(t)
	studentUserID := uuid.New()
	student := newTestStudent(t, account.StudentID, studentUserID)

	svc := newSavingsAccountServiceForTest(
		t,
		repository.RepositorySet{
			SavingsAccounts: &savingsAccountServiceFakeSavingsAccountRepository{
				getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) {
					return account, nil
				},
			},
			Students: &savingsAccountServiceFakeStudentRepository{
				getByIDFn: func(context.Context, uuid.UUID) (domain.Student, error) {
					return student, nil
				},
			},
		},
		&savingsAccountServiceFakeUnitOfWorkManager{},
	)

	got, err := svc.Get(context.Background(), GetSavingsAccountInput{
		Actor:            Actor{UserID: studentUserID, Role: domain.RoleSiswa},
		SavingsAccountID: account.ID,
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Account == nil || got.Account.ID != account.ID {
		t.Fatalf("Get() Account = %#v, want account %v", got.Account, account.ID)
	}
}

func TestSavingsAccountServiceGetStudentRejectsOtherStudent(t *testing.T) {
	account := newTestSavingsAccount(t)
	studentUserID := uuid.New()
	student := newTestStudent(t, account.StudentID, studentUserID)

	svc := newSavingsAccountServiceForTest(
		t,
		repository.RepositorySet{
			SavingsAccounts: &savingsAccountServiceFakeSavingsAccountRepository{
				getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) {
					return account, nil
				},
			},
			Students: &savingsAccountServiceFakeStudentRepository{
				getByIDFn: func(context.Context, uuid.UUID) (domain.Student, error) {
					return student, nil
				},
			},
		},
		&savingsAccountServiceFakeUnitOfWorkManager{},
	)

	_, err := svc.Get(context.Background(), GetSavingsAccountInput{
		Actor:            Actor{UserID: uuid.New(), Role: domain.RoleSiswa},
		SavingsAccountID: account.ID,
	})
	if !errors.Is(err, ErrScopeViolation) {
		t.Fatalf("Get() error = %v, want ErrScopeViolation", err)
	}
}

func TestSavingsAccountServiceGetWaliKelasInScope(t *testing.T) {
	account := newTestSavingsAccount(t)
	actorID := uuid.New()
	classID := uuid.New()

	repos := newTestScopeRepositories(
		&savingsAccountServiceFakeStudentClassHistoryRepository{
			listFn: func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.StudentClassHistory, error) {
				return []domain.StudentClassHistory{{
					StudentID: account.StudentID, AcademicYearID: account.AcademicYearID, ClassID: classID,
				}}, nil
			},
		},
		&savingsAccountServiceFakeTeacherAssignmentRepository{
			listByClassFn: func(context.Context, uuid.UUID, uuid.UUID, repository.ListOptions) ([]domain.TeacherClassAssignment, error) {
				return []domain.TeacherClassAssignment{{UserID: actorID, ClassID: classID, AcademicYearID: account.AcademicYearID}}, nil
			},
		},
	)
	repos.SavingsAccounts = &savingsAccountServiceFakeSavingsAccountRepository{
		getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
	}

	svc := newSavingsAccountServiceForTest(t, repos, &savingsAccountServiceFakeUnitOfWorkManager{})
	got, err := svc.Get(context.Background(), GetSavingsAccountInput{
		Actor: Actor{UserID: actorID, Role: domain.RoleWaliKelas}, SavingsAccountID: account.ID,
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Account == nil || got.Account.ID != account.ID {
		t.Fatalf("Get() Account = %#v, want account %v", got.Account, account.ID)
	}
}

func TestSavingsAccountServiceGetPropagatesNotFound(t *testing.T) {
	svc := newSavingsAccountServiceForTest(
		t,
		repository.RepositorySet{
			SavingsAccounts: &savingsAccountServiceFakeSavingsAccountRepository{
				getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) {
					return domain.SavingsAccount{}, repository.ErrNotFound
				},
			},
		},
		&savingsAccountServiceFakeUnitOfWorkManager{},
	)

	_, err := svc.Get(context.Background(), GetSavingsAccountInput{
		Actor:            Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
		SavingsAccountID: uuid.New(),
	})
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("Get() error = %v, want repository.ErrNotFound", err)
	}
}

func TestSavingsAccountServiceGetByStudentAndAcademicYearAdmin(t *testing.T) {
	account := newTestSavingsAccount(t)
	repo := &savingsAccountServiceFakeSavingsAccountRepository{
		getByStudentAndAcademicYearFn: func(context.Context, uuid.UUID, uuid.UUID) (domain.SavingsAccount, error) {
			return account, nil
		},
	}
	svc := newSavingsAccountServiceForTest(
		t,
		repository.RepositorySet{SavingsAccounts: repo},
		&savingsAccountServiceFakeUnitOfWorkManager{},
	)

	got, err := svc.GetByStudentAndAcademicYear(
		context.Background(),
		GetSavingsAccountByStudentAndAcademicYearInput{
			Actor:          Actor{UserID: uuid.New(), Role: domain.RoleAdmin},
			StudentID:      account.StudentID,
			AcademicYearID: account.AcademicYearID,
		},
	)
	if err != nil {
		t.Fatalf("GetByStudentAndAcademicYear() error = %v", err)
	}
	if got.Account == nil || got.Account.ID != account.ID {
		t.Fatalf("Account = %#v, want account %v", got.Account, account.ID)
	}
}

func TestSavingsAccountServiceGetByStudentAndAcademicYearStudentSelf(t *testing.T) {
	account := newTestSavingsAccount(t)
	studentUserID := uuid.New()
	student := newTestStudent(t, account.StudentID, studentUserID)

	svc := newSavingsAccountServiceForTest(
		t,
		repository.RepositorySet{
			Students: &savingsAccountServiceFakeStudentRepository{
				getByIDFn: func(context.Context, uuid.UUID) (domain.Student, error) { return student, nil },
			},
			SavingsAccounts: &savingsAccountServiceFakeSavingsAccountRepository{
				getByStudentAndAcademicYearFn: func(context.Context, uuid.UUID, uuid.UUID) (domain.SavingsAccount, error) {
					return account, nil
				},
			},
		},
		&savingsAccountServiceFakeUnitOfWorkManager{},
	)

	got, err := svc.GetByStudentAndAcademicYear(context.Background(), GetSavingsAccountByStudentAndAcademicYearInput{
		Actor: Actor{UserID: studentUserID, Role: domain.RoleSiswa}, StudentID: account.StudentID, AcademicYearID: account.AcademicYearID,
	})
	if err != nil {
		t.Fatalf("GetByStudentAndAcademicYear() error = %v", err)
	}
	if got.Account == nil || got.Account.ID != account.ID {
		t.Fatalf("Account = %#v, want account %v", got.Account, account.ID)
	}
}

func TestSavingsAccountServiceGetByStudentAndAcademicYearRejectsSiswaOtherStudent(t *testing.T) {
	account := newTestSavingsAccount(t)
	studentUserID := uuid.New()
	student := newTestStudent(t, account.StudentID, studentUserID)

	svc := newSavingsAccountServiceForTest(
		t,
		repository.RepositorySet{
			Students: &savingsAccountServiceFakeStudentRepository{
				getByIDFn: func(context.Context, uuid.UUID) (domain.Student, error) { return student, nil },
			},
			SavingsAccounts: &savingsAccountServiceFakeSavingsAccountRepository{},
		},
		&savingsAccountServiceFakeUnitOfWorkManager{},
	)

	_, err := svc.GetByStudentAndAcademicYear(context.Background(), GetSavingsAccountByStudentAndAcademicYearInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleSiswa}, StudentID: account.StudentID, AcademicYearID: account.AcademicYearID,
	})
	if !errors.Is(err, ErrScopeViolation) {
		t.Fatalf("GetByStudentAndAcademicYear() error = %v, want ErrScopeViolation", err)
	}
}

func TestSavingsAccountServiceGetBalance(t *testing.T) {
	account := newTestSavingsAccount(t)
	depositDate := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	transactions := []domain.Transaction{
		{ID: uuid.New(), SavingsAccountID: account.ID, Type: domain.TransactionDeposit, Amount: 100000, TransactionDate: depositDate, CreatedAt: depositDate, Status: domain.TransactionActive},
		{ID: uuid.New(), SavingsAccountID: account.ID, Type: domain.TransactionWithdrawal, Amount: 25000, TransactionDate: depositDate.AddDate(0, 0, 1), CreatedAt: depositDate.AddDate(0, 0, 1), Status: domain.TransactionActive},
		{ID: uuid.New(), SavingsAccountID: account.ID, Type: domain.TransactionDeposit, Amount: 10000, Status: domain.TransactionCancelled},
	}

	svc := newSavingsAccountServiceForTest(
		t,
		repository.RepositorySet{
			SavingsAccounts: &savingsAccountServiceFakeSavingsAccountRepository{
				getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
			},
			Transactions: &savingsAccountServiceFakeTransactionRepository{
				listActiveFn: func(context.Context, uuid.UUID) ([]domain.Transaction, error) { return transactions, nil },
			},
		},
		&savingsAccountServiceFakeUnitOfWorkManager{},
	)

	got, err := svc.GetBalance(context.Background(), GetSavingsAccountBalanceInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, SavingsAccountID: account.ID,
	})
	if err != nil {
		t.Fatalf("GetBalance() error = %v", err)
	}
	if got.Balance != 75000 {
		t.Fatalf("Balance = %v, want 75000", got.Balance)
	}
}

func TestSavingsAccountServiceBalanceDoesNotMixAcademicYears(t *testing.T) {
	first, second := newTestSavingsAccount(t), newTestSavingsAccount(t)
	studentID := uuid.New()
	first.StudentID, second.StudentID = studentID, studentID
	first.AcademicYearID, second.AcademicYearID = uuid.New(), uuid.New()
	accounts := map[uuid.UUID]domain.SavingsAccount{first.ID: first, second.ID: second}
	transactions := map[uuid.UUID][]domain.Transaction{
		first.ID:  {{ID: uuid.New(), SavingsAccountID: first.ID, Type: domain.TransactionDeposit, Amount: 120_000, Status: domain.TransactionActive}},
		second.ID: {{ID: uuid.New(), SavingsAccountID: second.ID, Type: domain.TransactionDeposit, Amount: 35_000, Status: domain.TransactionActive}},
	}
	repos := repository.RepositorySet{
		SavingsAccounts: &savingsAccountServiceFakeSavingsAccountRepository{getByIDFn: func(_ context.Context, id uuid.UUID) (domain.SavingsAccount, error) { return accounts[id], nil }},
		Transactions:    &savingsAccountServiceFakeTransactionRepository{listActiveFn: func(_ context.Context, id uuid.UUID) ([]domain.Transaction, error) { return transactions[id], nil }},
	}
	svc := newSavingsAccountServiceForTest(t, repos, &savingsAccountServiceFakeUnitOfWorkManager{})
	actor := Actor{UserID: uuid.New(), Role: domain.RoleAdmin}
	for _, tc := range []struct {
		account domain.SavingsAccount
		want    domain.Money
	}{{first, 120_000}, {second, 35_000}} {
		got, err := svc.GetBalance(context.Background(), GetSavingsAccountBalanceInput{Actor: actor, SavingsAccountID: tc.account.ID})
		if err != nil || got.Balance != tc.want {
			t.Fatalf("year %s balance=%d err=%v, want %d", tc.account.AcademicYearID, got.Balance, err, tc.want)
		}
	}
}

func TestSavingsAccountServiceGetBalancePropagatesTransactionError(t *testing.T) {
	wantErr := errors.New("list transactions failed")
	account := newTestSavingsAccount(t)

	svc := newSavingsAccountServiceForTest(
		t,
		repository.RepositorySet{
			SavingsAccounts: &savingsAccountServiceFakeSavingsAccountRepository{
				getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
			},
			Transactions: &savingsAccountServiceFakeTransactionRepository{
				listActiveFn: func(context.Context, uuid.UUID) ([]domain.Transaction, error) { return nil, wantErr },
			},
		},
		&savingsAccountServiceFakeUnitOfWorkManager{},
	)

	_, err := svc.GetBalance(context.Background(), GetSavingsAccountBalanceInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, SavingsAccountID: account.ID,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("GetBalance() error = %v, want %v", err, wantErr)
	}
}

func TestSavingsAccountServiceGetBalanceRejectsNegativeIntermediateBalance(t *testing.T) {
	account := newTestSavingsAccount(t)
	transactions := []domain.Transaction{
		{ID: uuid.New(), SavingsAccountID: account.ID, Type: domain.TransactionWithdrawal, Amount: 1000, Status: domain.TransactionActive},
	}

	svc := newSavingsAccountServiceForTest(
		t,
		repository.RepositorySet{
			SavingsAccounts: &savingsAccountServiceFakeSavingsAccountRepository{
				getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
			},
			Transactions: &savingsAccountServiceFakeTransactionRepository{
				listActiveFn: func(context.Context, uuid.UUID) ([]domain.Transaction, error) { return transactions, nil },
			},
		},
		&savingsAccountServiceFakeUnitOfWorkManager{},
	)

	_, err := svc.GetBalance(context.Background(), GetSavingsAccountBalanceInput{
		Actor: Actor{UserID: uuid.New(), Role: domain.RoleAdmin}, SavingsAccountID: account.ID,
	})
	if !errors.Is(err, domain.ErrNegativeBalance) {
		t.Fatalf("GetBalance() error = %v, want domain.ErrNegativeBalance", err)
	}
}

func TestSavingsAccountServiceGetBalanceStudentSelf(t *testing.T) {
	account := newTestSavingsAccount(t)
	studentUserID := uuid.New()
	student := newTestStudent(t, account.StudentID, studentUserID)

	svc := newSavingsAccountServiceForTest(
		t,
		repository.RepositorySet{
			SavingsAccounts: &savingsAccountServiceFakeSavingsAccountRepository{
				getByIDFn: func(context.Context, uuid.UUID) (domain.SavingsAccount, error) { return account, nil },
			},
			Students: &savingsAccountServiceFakeStudentRepository{
				getByIDFn: func(context.Context, uuid.UUID) (domain.Student, error) { return student, nil },
			},
			Transactions: &savingsAccountServiceFakeTransactionRepository{
				listActiveFn: func(context.Context, uuid.UUID) ([]domain.Transaction, error) { return nil, nil },
			},
		},
		&savingsAccountServiceFakeUnitOfWorkManager{},
	)

	got, err := svc.GetBalance(context.Background(), GetSavingsAccountBalanceInput{
		Actor: Actor{UserID: studentUserID, Role: domain.RoleSiswa}, SavingsAccountID: account.ID,
	})
	if err != nil {
		t.Fatalf("GetBalance() error = %v", err)
	}
	if got.Balance != 0 {
		t.Fatalf("Balance = %v, want 0", got.Balance)
	}
}

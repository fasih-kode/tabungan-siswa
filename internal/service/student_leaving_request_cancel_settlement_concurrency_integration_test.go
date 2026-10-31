package service

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/google/uuid"
)

func TestLeavingRequestCancelVsInternalSettlement(t *testing.T) {
	t.Run("settlement commit lebih dahulu", func(t *testing.T) {
		f := newInternalLeavingCancelFixture(t)
		gate := newInternalCancelGate()
		settlementService := &settlementService{deps: Dependencies{Repositories: f.repos, UOW: internalCancelBarrierManager{UnitOfWorkManager: f.manager, mode: "settlement-account", gate: gate}}}
		settleResult := make(chan error, 1)
		go func() {
			_, err := settlementService.settleForStudentLeavingRequest(context.Background(), settleForStudentLeavingRequestInput{
				Actor: f.waliActor, StudentLeavingRequestID: f.request.ID, SavingsAccountID: f.account.ID, ExpectedBalance: 50_000,
			})
			settleResult <- err
		}()
		waitInternalCancelGate(t, gate)
		blockerPID := internalCancelIdleTransactionPID(t, f.db)
		cancelResult := make(chan error, 1)
		go func() {
			_, err := f.cancelService.Cancel(context.Background(), CancelStudentLeavingRequestInput{Actor: f.adminActor, StudentLeavingRequestID: f.request.ID, Reason: "uji settlement lebih dahulu"})
			cancelResult <- err
		}()
		waitInternalCancelBlocker(t, f.db, blockerPID)
		gate.release()
		if err := <-settleResult; err != nil {
			t.Fatalf("settleForStudentLeavingRequest(): %v", err)
		}
		if err := <-cancelResult; !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("Cancel() error=%v, want incomplete handover rejection", err)
		}
		account, err := f.repos.SavingsAccounts.GetByID(context.Background(), f.account.ID)
		if err != nil || !account.IsSettled() {
			t.Fatalf("account after settlement=%+v err=%v", account, err)
		}
		settlements, err := f.repos.Settlements.ListByStudentLeavingRequest(context.Background(), f.request.ID)
		if err != nil || len(settlements) != 1 {
			t.Fatalf("linked settlements=%d err=%v", len(settlements), err)
		}
	})

	t.Run("Cancel commit lebih dahulu", func(t *testing.T) {
		f := newInternalLeavingCancelFixture(t)
		gate := newInternalCancelGate()
		cancelService, err := NewStudentLeavingRequestService(Dependencies{Repositories: f.repos, UOW: internalCancelBarrierManager{UnitOfWorkManager: f.manager, mode: "cancel-finance", gate: gate}})
		if err != nil {
			t.Fatal(err)
		}
		cancelResult := make(chan error, 1)
		go func() {
			_, cancelErr := cancelService.Cancel(context.Background(), CancelStudentLeavingRequestInput{Actor: f.adminActor, StudentLeavingRequestID: f.request.ID, Reason: "uji cancel lebih dahulu"})
			cancelResult <- cancelErr
		}()
		waitInternalCancelGate(t, gate)
		blockerPID := internalCancelIdleTransactionPID(t, f.db)
		settleResult := make(chan error, 1)
		go func() {
			_, settleErr := f.settlementService.settleForStudentLeavingRequest(context.Background(), settleForStudentLeavingRequestInput{
				Actor: f.waliActor, StudentLeavingRequestID: f.request.ID, SavingsAccountID: f.account.ID, ExpectedBalance: 50_000,
			})
			settleResult <- settleErr
		}()
		waitInternalCancelBlocker(t, f.db, blockerPID)
		gate.release()
		if err := <-cancelResult; err != nil {
			t.Fatalf("Cancel(): %v", err)
		}
		if err := <-settleResult; !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("settleForStudentLeavingRequest() error=%v, want cancelled-process rejection", err)
		}
		account, err := f.repos.SavingsAccounts.GetByID(context.Background(), f.account.ID)
		if err != nil || !account.IsOpen() {
			t.Fatalf("account after rejected settlement=%+v err=%v", account, err)
		}
		settlements, err := f.repos.Settlements.ListByStudentLeavingRequest(context.Background(), f.request.ID)
		if err != nil || len(settlements) != 0 {
			t.Fatalf("settlements after rejected settlement=%d err=%v", len(settlements), err)
		}
	})
}

type internalLeavingCancelFixture struct {
	db                *sql.DB
	repos             repository.RepositorySet
	manager           repository.UnitOfWorkManager
	adminActor        Actor
	waliActor         Actor
	student           domain.Student
	year              domain.AcademicYear
	class             domain.Class
	account           domain.SavingsAccount
	request           domain.StudentLeavingRequest
	cancelService     StudentLeavingRequestService
	settlementService *settlementService
}

func newInternalLeavingCancelFixture(t *testing.T) internalLeavingCancelFixture {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if err := database.ValidateTestDatabaseDSN(dsn); err != nil {
		t.Fatalf("unsafe integration DSN: %v", err)
	}
	db, err := database.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close(db) })
	var adminID, waliID, studentID, yearID, classID, accountID uuid.UUID
	t.Cleanup(func() {
		cleanupInternalCancelFixture(db, studentID, yearID, classID, accountID, adminID, waliID)
	})
	ctx := context.Background()
	var version int
	var dirty bool
	if err := db.QueryRowContext(ctx, `SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty); err != nil {
		t.Fatal(err)
	}
	if version != 17 || dirty {
		t.Skipf("requires clean schema v17; version=%d dirty=%t", version, dirty)
	}
	repos := postgres.NewRepositorySet(db)
	manager := postgres.NewUnitOfWorkManager(db)
	admin := newInternalCancelUser(t, repos, domain.RoleAdmin)
	adminID = admin.ID
	wali := newInternalCancelUser(t, repos, domain.RoleWaliKelas)
	waliID = wali.ID
	student, err := domain.NewStudent("Siswa cancel settlement "+uuid.NewString(), nil, stringPointer("nis-"+uuid.NewString()), stringPointer("nisn-"+uuid.NewString()))
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.Students.Create(ctx, student); err != nil {
		t.Fatal(err)
	}
	studentID = student.ID
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}
	today := time.Now().In(location)
	startYear := today.Year()
	if today.Month() < time.July {
		startYear--
	}
	year, err := domain.NewAcademicYear("Tahun cancel settlement "+uuid.NewString(), time.Date(startYear, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(startYear+1, 7, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.AcademicYears.Create(ctx, year); err != nil {
		t.Fatal(err)
	}
	yearID = year.ID
	class, err := domain.NewClass("Kelas cancel settlement "+uuid.NewString(), 7)
	if err != nil {
		t.Fatal(err)
	}
	class.AcademicYearID, class.EffectiveDate = year.ID, year.StartDate
	if err := repos.Classes.Create(ctx, class); err != nil {
		t.Fatal(err)
	}
	classID = class.ID
	period, err := domain.NewDateRange(year.StartDate, &year.EndDate)
	if err != nil {
		t.Fatal(err)
	}
	history, err := domain.NewStudentClassHistory(student.ID, year.ID, class.ID, period)
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.StudentClassHistories.Create(ctx, history); err != nil {
		t.Fatal(err)
	}
	assignment, err := domain.NewTeacherClassAssignment(wali.ID, year.ID, class.ID, period)
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.TeacherClassAssignments.Create(ctx, assignment); err != nil {
		t.Fatal(err)
	}
	account, err := domain.NewSavingsAccount(student.ID, year.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.SavingsAccounts.Create(ctx, account); err != nil {
		t.Fatal(err)
	}
	accountID = account.ID
	deposit, err := domain.NewDeposit(account.ID, 50_000, dateOnlyUTC(today), wali.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.Transactions.Create(ctx, deposit); err != nil {
		t.Fatal(err)
	}
	effectiveDate := dateOnlyUTC(today)
	cancelService, err := NewStudentLeavingRequestService(Dependencies{Repositories: repos, UOW: manager})
	if err != nil {
		t.Fatal(err)
	}
	started, err := cancelService.Start(ctx, StartStudentLeavingRequestInput{Actor: Actor{UserID: admin.ID, Role: domain.RoleAdmin}, StudentID: student.ID, AcademicYearID: year.ID, EffectiveLeavingDate: effectiveDate})
	if err != nil {
		t.Fatal(err)
	}
	return internalLeavingCancelFixture{
		db: db, repos: repos, manager: manager, adminActor: Actor{UserID: admin.ID, Role: domain.RoleAdmin}, waliActor: Actor{UserID: wali.ID, Role: domain.RoleWaliKelas},
		student: student, year: year, class: class, account: account, request: *started.Request, cancelService: cancelService,
		settlementService: &settlementService{deps: Dependencies{Repositories: repos, UOW: manager}},
	}
}

func cleanupInternalCancelFixture(db *sql.DB, studentID, yearID, classID, accountID, adminID, waliID uuid.UUID) {
	ctx := context.Background()
	statements := []struct {
		query string
		args  []any
	}{
		{`DELETE FROM audit_logs WHERE entity_id=$1 OR actor_user_id IN ($2,$3)`, []any{studentID, adminID, waliID}},
		{`DELETE FROM settlement_cash_movements WHERE savings_settlement_id IN (SELECT id FROM savings_settlements WHERE savings_account_id=$1)`, []any{accountID}},
		{`DELETE FROM settlement_reopen_requests WHERE savings_settlement_id IN (SELECT id FROM savings_settlements WHERE savings_account_id=$1)`, []any{accountID}},
		{`DELETE FROM savings_settlements WHERE savings_account_id=$1`, []any{accountID}},
		{`DELETE FROM transactions WHERE savings_account_id=$1`, []any{accountID}},
		{`DELETE FROM student_leaving_requests WHERE student_id=$1`, []any{studentID}},
		{`DELETE FROM teacher_class_assignments WHERE user_id IN ($1,$2) OR academic_year_id=$3`, []any{waliID, adminID, yearID}},
		{`DELETE FROM student_class_histories WHERE student_id=$1`, []any{studentID}},
		{`DELETE FROM savings_accounts WHERE student_id=$1`, []any{studentID}},
		{`DELETE FROM classes WHERE id=$1`, []any{classID}},
		{`DELETE FROM students WHERE id=$1`, []any{studentID}},
		{`DELETE FROM users WHERE id IN ($1,$2)`, []any{adminID, waliID}},
		{`DELETE FROM academic_years WHERE id=$1`, []any{yearID}},
	}
	for _, statement := range statements {
		_, _ = db.ExecContext(ctx, statement.query, statement.args...)
	}
}

func newInternalCancelUser(t *testing.T, repos repository.RepositorySet, role domain.UserRole) domain.User {
	t.Helper()
	user, err := domain.NewUser("cancel-"+uuid.NewString(), "test-hash", role)
	if err != nil {
		t.Fatal(err)
	}
	if err := repos.Users.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	return user
}

func stringPointer(value string) *string { return &value }

type internalCancelGate struct {
	reached   chan struct{}
	releaseCh chan struct{}
	once      sync.Once
}

func newInternalCancelGate() *internalCancelGate {
	return &internalCancelGate{reached: make(chan struct{}), releaseCh: make(chan struct{})}
}
func (g *internalCancelGate) pause() { g.once.Do(func() { close(g.reached); <-g.releaseCh }) }
func (g *internalCancelGate) release() {
	select {
	case <-g.releaseCh:
	default:
		close(g.releaseCh)
	}
}

type internalCancelBarrierManager struct {
	repository.UnitOfWorkManager
	mode string
	gate *internalCancelGate
}

func (m internalCancelBarrierManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	uow, err := m.UnitOfWorkManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := uow.Repositories()
	switch m.mode {
	case "settlement-account":
		repos.SavingsAccounts = internalCancelPauseAccountRepository{SavingsAccountRepository: repos.SavingsAccounts, gate: m.gate}
	case "cancel-finance":
		repos.StudentLeavingRequests = internalCancelPauseRequestRepository{StudentLeavingRequestRepository: repos.StudentLeavingRequests, gate: m.gate}
	}
	return internalCancelBarrierUOW{UnitOfWork: uow, repos: repos}, nil
}

type internalCancelBarrierUOW struct {
	repository.UnitOfWork
	repos repository.RepositorySet
}

func (u internalCancelBarrierUOW) Repositories() repository.RepositorySet { return u.repos }

type internalCancelPauseAccountRepository struct {
	repository.SavingsAccountRepository
	gate *internalCancelGate
}

func (r internalCancelPauseAccountRepository) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.SavingsAccount, error) {
	account, err := r.SavingsAccountRepository.GetByIDForUpdate(ctx, id)
	if err == nil {
		r.gate.pause()
	}
	return account, err
}

type internalCancelPauseRequestRepository struct {
	repository.StudentLeavingRequestRepository
	gate *internalCancelGate
}

func (r internalCancelPauseRequestRepository) ListFinancialLinkIssues(ctx context.Context, id uuid.UUID) ([]repository.StudentLeavingFinancialLinkIssue, error) {
	issues, err := r.StudentLeavingRequestRepository.ListFinancialLinkIssues(ctx, id)
	if err == nil {
		r.gate.pause()
	}
	return issues, err
}

func waitInternalCancelGate(t *testing.T, gate *internalCancelGate) {
	t.Helper()
	select {
	case <-gate.reached:
	case <-time.After(8 * time.Second):
		t.Fatal("timeout menunggu barrier")
	}
}
func internalCancelIdleTransactionPID(t *testing.T, db *sql.DB) int {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		var pid int
		err := db.QueryRow(`SELECT pid FROM pg_stat_activity WHERE datname=current_database() AND state='idle in transaction' AND xact_start IS NOT NULL AND pid<>pg_backend_pid() ORDER BY xact_start DESC LIMIT 1`).Scan(&pid)
		if err == nil {
			return pid
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("backend transaksi tidak ditemukan")
	return 0
}
func waitInternalCancelBlocker(t *testing.T, db *sql.DB, blockerPID int) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		var blocked bool
		if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND pid<>pg_backend_pid() AND $1=ANY(pg_blocking_pids(pid)))`, blockerPID).Scan(&blocked); err != nil {
			t.Fatal(err)
		}
		if blocked {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("tidak melihat backend diblokir oleh %d", blockerPID)
}

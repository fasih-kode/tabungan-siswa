package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestStudentReactivateIntegrationValidatesPlacementAndStatus(t *testing.T) {
	f := newStudentPlacementFixture(t)
	ctx := context.Background()
	created, err := f.students.Create(ctx, f.createInput("reactivate-valid"))
	if err != nil {
		t.Fatal(err)
	}
	studentID := created.Student.ID
	leaveDate := f.class.EffectiveDate.AddDate(0, 0, 60)
	input := service.ReactivateStudentInput{
		Actor:          f.actor,
		StudentID:      studentID,
		AcademicYearID: f.year.ID,
		ClassID:        f.class.ID,
		EffectiveDate:  leaveDate.AddDate(0, 0, 1),
	}

	if _, err := f.students.Reactivate(ctx, input); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("Reactivate(active student) error = %v, want invalid transition", err)
	}
	if _, err := f.students.Leave(ctx, service.LeaveStudentInput{
		Actor: f.actor, StudentID: studentID, AcademicYearID: f.year.ID, LeavingDate: leaveDate,
	}); err != nil {
		t.Fatalf("Leave() error = %v", err)
	}

	otherYear := uuid.New()
	invalidInputs := []struct {
		name string
		edit func(*service.ReactivateStudentInput)
		want error
	}{
		{name: "tahun tidak ditemukan", edit: func(in *service.ReactivateStudentInput) { in.AcademicYearID = otherYear }, want: repository.ErrNotFound},
		{name: "kelas dari tahun lain", edit: func(in *service.ReactivateStudentInput) { in.ClassID = f.otherClass.ID }, want: domain.ErrInvalidValue},
		{name: "tanggal sebelum kelas efektif", edit: func(in *service.ReactivateStudentInput) { in.EffectiveDate = f.class.EffectiveDate.AddDate(0, 0, -1) }, want: domain.ErrInvalidDateRange},
		{name: "tanggal pada akhir tahun", edit: func(in *service.ReactivateStudentInput) { in.EffectiveDate = f.year.EndDate }, want: domain.ErrInvalidDateRange},
	}
	for _, tc := range invalidInputs {
		t.Run(tc.name, func(t *testing.T) {
			invalid := input
			tc.edit(&invalid)
			if _, err := f.students.Reactivate(ctx, invalid); !errors.Is(err, tc.want) {
				t.Fatalf("Reactivate() error = %v, want %v", err, tc.want)
			}
			assertStudentPlacementUnchanged(t, ctx, f, studentID, domain.StudentLeft, 1)
		})
	}

	output, err := f.students.Reactivate(ctx, input)
	if err != nil {
		t.Fatalf("Reactivate(valid input) error = %v", err)
	}
	if output.Student == nil || output.Student.Status != domain.StudentActive {
		t.Fatalf("Reactivate() student = %+v, want ACTIVE", output.Student)
	}
	if output.History == nil || output.History.ClassID != f.class.ID || !output.History.Period.From.Equal(input.EffectiveDate) {
		t.Fatalf("Reactivate() history = %+v", output.History)
	}
	assertStudentPlacementUnchanged(t, ctx, f, studentID, domain.StudentActive, 2)
}

func TestStudentReactivateIntegrationRollsBackHistoryWhenStudentUpdateFails(t *testing.T) {
	f := newStudentPlacementFixture(t)
	ctx := context.Background()
	created, err := f.students.Create(ctx, f.createInput("reactivate-rollback"))
	if err != nil {
		t.Fatal(err)
	}
	studentID := created.Student.ID
	leaveDate := f.class.EffectiveDate.AddDate(0, 0, 60)
	if _, err := f.students.Leave(ctx, service.LeaveStudentInput{
		Actor: f.actor, StudentID: studentID, AcademicYearID: f.year.ID, LeavingDate: leaveDate,
	}); err != nil {
		t.Fatalf("Leave() error = %v", err)
	}

	failure := errors.New("injected student update failure")
	manager := failingStudentUpdateManager{UnitOfWorkManager: f.manager, failure: failure}
	svc, err := service.NewStudentService(service.Dependencies{Repositories: f.repos, UOW: manager})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Reactivate(ctx, service.ReactivateStudentInput{
		Actor: f.actor, StudentID: studentID, AcademicYearID: f.year.ID, ClassID: f.class.ID,
		EffectiveDate: leaveDate.AddDate(0, 0, 1),
	}); !errors.Is(err, failure) {
		t.Fatalf("Reactivate() error = %v, want injected error", err)
	}
	assertStudentPlacementUnchanged(t, ctx, f, studentID, domain.StudentLeft, 1)

	var auditCount int
	if err := f.db.QueryRowContext(ctx,
		`SELECT count(*) FROM audit_logs WHERE entity_id = $1`, studentID,
	).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 0 {
		t.Fatalf("failed Reactivate() left %d audit rows for student", auditCount)
	}
}

type failingStudentUpdateRepository struct {
	repository.StudentRepository
	failure error
}

func (r failingStudentUpdateRepository) Update(context.Context, domain.Student) error {
	return r.failure
}

type failingStudentUpdateUnitOfWork struct {
	repository.UnitOfWork
	repos repository.RepositorySet
}

func (u failingStudentUpdateUnitOfWork) Repositories() repository.RepositorySet { return u.repos }

type failingStudentUpdateManager struct {
	repository.UnitOfWorkManager
	failure error
}

func (m failingStudentUpdateManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	uow, err := m.UnitOfWorkManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := uow.Repositories()
	repos.Students = failingStudentUpdateRepository{StudentRepository: repos.Students, failure: m.failure}
	return failingStudentUpdateUnitOfWork{UnitOfWork: uow, repos: repos}, nil
}

func assertStudentPlacementUnchanged(
	t *testing.T,
	ctx context.Context,
	f studentPlacementFixture,
	studentID uuid.UUID,
	wantStatus domain.StudentStatus,
	wantHistories int,
) {
	t.Helper()
	student, err := f.repos.Students.GetByID(ctx, studentID)
	if err != nil {
		t.Fatal(err)
	}
	if student.Status != wantStatus {
		t.Fatalf("student status = %s, want %s", student.Status, wantStatus)
	}
	histories, err := f.repos.StudentClassHistories.ListByStudentAndAcademicYear(
		ctx, studentID, f.year.ID, repository.ListOptions{Limit: 10},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(histories) != wantHistories {
		t.Fatalf("history count = %d, want %d: %+v", len(histories), wantHistories, histories)
	}
}

func TestStudentReactivateIntegrationSerializesDepositAndWithdrawal(t *testing.T) {
	for _, transactionType := range []string{"setoran", "penarikan"} {
		t.Run(transactionType, func(t *testing.T) {
			f := newStudentPlacementFixture(t)
			ctx := context.Background()
			wali := newServiceScopeUser(t, domain.RoleWaliKelas)
			if err := f.repos.Users.Create(ctx, wali); err != nil {
				t.Fatal(err)
			}
			assignmentPeriod, err := domain.NewDateRange(f.year.StartDate, &f.year.EndDate)
			if err != nil {
				t.Fatal(err)
			}
			assignment, err := domain.NewTeacherClassAssignment(wali.ID, f.year.ID, f.class.ID, assignmentPeriod)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.repos.TeacherClassAssignments.Create(ctx, assignment); err != nil {
				t.Fatal(err)
			}
			created, err := f.students.Create(ctx, f.createInput("reactivate-race-"+uuid.NewString()))
			if err != nil {
				t.Fatal(err)
			}
			studentID := created.Student.ID
			account, err := domain.NewSavingsAccount(studentID, f.year.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.repos.SavingsAccounts.Create(ctx, account); err != nil {
				t.Fatal(err)
			}
			transactionService, err := service.NewTransactionService(service.Dependencies{
				Repositories: f.repos,
				UOW:          f.manager,
			})
			if err != nil {
				t.Fatal(err)
			}
			waliActor := service.Actor{UserID: wali.ID, Role: domain.RoleWaliKelas}
			transactionDate := f.class.EffectiveDate.AddDate(0, 0, 20)
			if transactionType == "penarikan" {
				if _, err := transactionService.Deposit(ctx, service.DepositInput{
					Actor: waliActor, SavingsAccountID: account.ID, Amount: domain.Money(50_000), TransactionDate: transactionDate,
				}); err != nil {
					t.Fatalf("initial Deposit() error = %v", err)
				}
			}
			leaveDate := f.class.EffectiveDate.AddDate(0, 0, 60)
			if _, err := f.students.Leave(ctx, service.LeaveStudentInput{
				Actor: f.actor, StudentID: studentID, AcademicYearID: f.year.ID, LeavingDate: leaveDate,
			}); err != nil {
				t.Fatalf("Leave() error = %v", err)
			}

			blockerConn, err := f.db.Conn(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer blockerConn.Close()
			blockerTx, err := blockerConn.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer blockerTx.Rollback()
			var blockerPID int
			if err := blockerConn.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&blockerPID); err != nil {
				t.Fatal(err)
			}
			if _, err := blockerTx.ExecContext(ctx, "SELECT id FROM students WHERE id = $1 FOR UPDATE", studentID); err != nil {
				t.Fatal(err)
			}

			reactivateCtx, cancelReactivate := context.WithTimeout(ctx, 20*time.Second)
			defer cancelReactivate()
			reactivateDone := make(chan error, 1)
			go func() {
				_, err := f.students.Reactivate(reactivateCtx, service.ReactivateStudentInput{
					Actor: f.actor, StudentID: studentID, AcademicYearID: f.year.ID, ClassID: f.class.ID,
					EffectiveDate: leaveDate.AddDate(0, 0, 1),
				})
				reactivateDone <- err
			}()
			reactivatePID := waitForBlockedBackend(t, f.db, blockerPID, "%FROM students%FOR UPDATE%")

			transactionCtx, cancelTransaction := context.WithTimeout(ctx, 20*time.Second)
			defer cancelTransaction()
			transactionDone := make(chan error, 1)
			go func() {
				var err error
				if transactionType == "setoran" {
					_, err = transactionService.Deposit(transactionCtx, service.DepositInput{
						Actor: waliActor, SavingsAccountID: account.ID, Amount: domain.Money(10_000),
						TransactionDate: leaveDate.AddDate(0, 0, 1),
					})
				} else {
					_, err = transactionService.Withdrawal(transactionCtx, service.WithdrawalInput{
						Actor: waliActor, SavingsAccountID: account.ID, Amount: domain.Money(10_000),
						TransactionDate: leaveDate.AddDate(0, 0, 1),
					})
				}
				transactionDone <- err
			}()
			waitForBlockedBackend(t, f.db, reactivatePID, "%FROM academic_years%FOR SHARE%")

			if err := blockerTx.Commit(); err != nil {
				t.Fatalf("release student lock: %v", err)
			}
			if err := <-reactivateDone; err != nil {
				t.Fatalf("Reactivate() error = %v", err)
			}
			if err := <-transactionDone; err != nil {
				t.Fatalf("concurrent %s error = %v", transactionType, err)
			}

			student, err := f.repos.Students.GetByID(ctx, studentID)
			if err != nil || student.Status != domain.StudentActive {
				t.Fatalf("final student status = %s, error = %v", student.Status, err)
			}
			transactions, err := f.repos.Transactions.ListActiveBySavingsAccount(ctx, account.ID)
			if err != nil {
				t.Fatal(err)
			}
			wantCount := 1
			wantBalance := domain.Money(10_000)
			if transactionType == "penarikan" {
				wantCount = 2
				wantBalance = domain.Money(40_000)
			}
			if len(transactions) != wantCount {
				t.Fatalf("active transaction count = %d, want %d", len(transactions), wantCount)
			}
			balance, err := domain.CalculateBalance(transactions)
			if err != nil || balance.Amount != wantBalance {
				t.Fatalf("balance = %d, error = %v; want %d", balance.Amount, err, wantBalance)
			}
			if err := assertTransactionAudits(ctx, f.db, transactions); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func waitForBlockedBackend(t *testing.T, db *sql.DB, blockerPID int, queryPattern string) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		var pid int
		err := db.QueryRowContext(ctx, `
			SELECT pid
			FROM pg_stat_activity
			WHERE datname = current_database()
			  AND wait_event_type = 'Lock'
			  AND query ILIKE $2
			  AND $1 = ANY(pg_blocking_pids(pid))
			ORDER BY query_start
			LIMIT 1`, blockerPID, queryPattern).Scan(&pid)
		if err == nil {
			return pid
		}
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("observe blocked backend: %v", err)
		}
		if ctx.Err() != nil {
			t.Fatalf("timed out waiting for blocked query %q behind pid %d", queryPattern, blockerPID)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func assertTransactionAudits(ctx context.Context, db *sql.DB, transactions []domain.Transaction) error {
	for _, transaction := range transactions {
		var count int
		if err := db.QueryRowContext(ctx,
			`SELECT count(*) FROM audit_logs WHERE entity_type = $1 AND entity_id = $2`,
			domain.AuditEntityTransaction, transaction.ID,
		).Scan(&count); err != nil {
			return err
		}
		if count != 1 {
			return errors.New("transaction must have exactly one audit record")
		}
	}
	return nil
}

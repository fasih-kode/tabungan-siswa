package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/google/uuid"
)

func TestReportingRepositoryUsesActiveLedgerAndActorScope(t *testing.T) {
	db := openTestDatabase(t)
	ctx := context.Background()
	fixture := newSavingsAccountFixture(t, db)

	class := newTestClass(t, "Reporting "+uuid.NewString(), 4)
	class.AcademicYearID = fixture.academicYear.ID
	class.EffectiveDate = fixture.academicYear.StartDate
	if err := postgres.NewClassRepository(db).Create(ctx, class); err != nil {
		t.Fatalf("create class: %v", err)
	}
	t.Cleanup(func() { deleteTestClass(t, db, class.ID) })

	history := newTestStudentClassHistory(t, fixture.student.ID, fixture.academicYear.ID, class.ID, fixture.academicYear.StartDate, fixture.academicYear.EndDate)
	if err := postgres.NewStudentClassHistoryRepository(db).Create(ctx, history); err != nil {
		t.Fatalf("create placement: %v", err)
	}
	t.Cleanup(func() {
		_, err := db.ExecContext(ctx, "DELETE FROM student_class_histories WHERE id=$1", history.ID)
		if err != nil {
			t.Errorf("delete placement: %v", err)
		}
	})

	account, err := domain.NewSavingsAccount(fixture.student.ID, fixture.academicYear.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := postgres.NewSavingsAccountRepository(db).Create(ctx, account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	t.Cleanup(func() {
		_, err := db.ExecContext(ctx, "DELETE FROM savings_accounts WHERE id=$1", account.ID)
		if err != nil {
			t.Errorf("delete account: %v", err)
		}
	})

	active, _ := domain.NewDeposit(account.ID, 100_000, time.Now(), fixture.user.ID)
	cancelled, _ := domain.NewWithdrawal(account.ID, 25_000, time.Now(), fixture.user.ID)
	if err := postgres.NewTransactionRepository(db).Create(ctx, active); err != nil {
		t.Fatalf("create active transaction: %v", err)
	}
	t.Cleanup(func() {
		_, err := db.ExecContext(ctx, "DELETE FROM transactions WHERE id=$1", active.ID)
		if err != nil {
			t.Errorf("delete active transaction: %v", err)
		}
	})
	cancelled.Status = domain.TransactionCancelled
	if err := postgres.NewTransactionRepository(db).Create(ctx, cancelled); err != nil {
		t.Fatalf("create cancelled transaction: %v", err)
	}
	t.Cleanup(func() {
		_, err := db.ExecContext(ctx, "DELETE FROM transactions WHERE id=$1", cancelled.ID)
		if err != nil {
			t.Errorf("delete cancelled transaction: %v", err)
		}
	})

	reporting := postgres.NewReportingRepository(db)
	got, err := reporting.Dashboard(ctx, fixture.academicYear.ID, fixture.user.ID, domain.RoleAdmin)
	if err != nil {
		t.Fatalf("Dashboard(): %v", err)
	}
	if got.Students != 1 || got.Classes != 1 || got.Accounts != 1 || got.Balance != 100_000 || got.Deposits != 100_000 || got.Withdrawals != 0 {
		t.Fatalf("Dashboard() = %+v; cancelled transaction must not count", got)
	}
	students, err := reporting.StudentReport(ctx, fixture.academicYear.ID, fixture.user.ID, domain.RoleAdmin)
	if err != nil {
		t.Fatalf("StudentReport(): %v", err)
	}
	if len(students) != 1 || students[0].AccountID != account.ID || students[0].Balance != 100_000 {
		t.Fatalf("StudentReport() = %+v", students)
	}
	classes, err := reporting.ClassReport(ctx, fixture.academicYear.ID, fixture.user.ID, domain.RoleAdmin)
	if err != nil {
		t.Fatalf("ClassReport(): %v", err)
	}
	if len(classes) != 1 || classes[0].Students != 1 || classes[0].Balance != 100_000 {
		t.Fatalf("ClassReport() = %+v", classes)
	}
	settlements, err := reporting.SettlementReport(ctx, fixture.academicYear.ID, fixture.user.ID, domain.RoleAdmin)
	if err != nil {
		t.Fatalf("SettlementReport(): %v", err)
	}
	if len(settlements) != 1 || settlements[0].AccountID != account.ID || settlements[0].Balance != 100_000 || settlements[0].SettlementState != "OUTSTANDING" {
		t.Fatalf("SettlementReport() = %+v", settlements)
	}

	// Role scope is applied inside SQL: the student actor for a different user
	// cannot see this student's row even though the caller supplies the same year.
	other, err := reporting.StudentReport(ctx, fixture.academicYear.ID, uuid.New(), domain.RoleSiswa)
	if err != nil {
		t.Fatalf("scoped StudentReport(): %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("student scope returned rows: %+v", other)
	}
	teacher, err := domain.NewUser("report-wali-"+uuid.NewString(), "test-password-hash", domain.RoleWaliKelas)
	if err != nil {
		t.Fatal(err)
	}
	if err := postgres.NewUserRepository(db).Create(ctx, teacher); err != nil {
		t.Fatalf("create report Wali: %v", err)
	}
	t.Cleanup(func() { deleteTestUser(t, db, teacher.ID) })
	withoutAssignment, err := reporting.StudentReport(ctx, fixture.academicYear.ID, teacher.ID, domain.RoleWaliKelas)
	if err != nil {
		t.Fatalf("unassigned Wali StudentReport(): %v", err)
	}
	if len(withoutAssignment) != 0 {
		t.Fatalf("unassigned Wali saw rows: %+v", withoutAssignment)
	}
	assignment := newTestTeacherClassAssignment(t, teacher.ID, fixture.academicYear.ID, class.ID, fixture.academicYear.StartDate, fixture.academicYear.EndDate)
	if err := postgres.NewTeacherClassAssignmentRepository(db).Create(ctx, assignment); err != nil {
		t.Fatalf("assign report Wali: %v", err)
	}
	t.Cleanup(func() {
		_, err := db.ExecContext(ctx, "DELETE FROM teacher_class_assignments WHERE id=$1", assignment.ID)
		if err != nil {
			t.Errorf("delete report Wali assignment: %v", err)
		}
	})
	assigned, err := reporting.StudentReport(ctx, fixture.academicYear.ID, teacher.ID, domain.RoleWaliKelas)
	if err != nil {
		t.Fatalf("assigned Wali StudentReport(): %v", err)
	}
	if len(assigned) != 1 || assigned[0].StudentID != fixture.student.ID {
		t.Fatalf("assigned Wali report = %+v", assigned)
	}
}

package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/platform/database"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/google/uuid"
)

type leavingSettlementFixture struct {
	db      *sql.DB
	repos   repository.RepositorySet
	manager repository.UnitOfWorkManager
	service *settlementService

	wali       domain.User
	priorYear  domain.AcademicYear
	year       domain.AcademicYear
	class      domain.Class
	student    domain.Student
	account    domain.SavingsAccount
	request    domain.StudentLeavingRequest
	assignment domain.TeacherClassAssignment
	history    domain.StudentClassHistory
}

func newLeavingSettlementFixture(t *testing.T) *leavingSettlementFixture {
	t.Helper()
	ctx := context.Background()
	dsn := os.Getenv("DATABASE_URL")
	if err := database.ValidateTestDatabaseDSN(dsn); err != nil {
		t.Fatalf("konfigurasi database tes tidak aman: %v", err)
	}
	db, err := database.Open(dsn)
	if err != nil {
		t.Fatalf("buka database tes: %v", err)
	}
	if err := database.Ping(ctx, db); err != nil {
		_ = db.Close()
		t.Fatalf("hubungkan database tes: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("tutup database tes: %v", err)
		}
	})
	var version int
	var dirty bool
	if err := db.QueryRowContext(ctx, `SELECT version, dirty FROM schema_migrations`).Scan(&version, &dirty); err != nil {
		t.Fatalf("baca versi migration: %v", err)
	}
	if version != 16 || dirty {
		t.Skipf("tes settlement proses pindah membutuhkan schema 16|false; tersedia %d|%t", version, dirty)
	}

	f := &leavingSettlementFixture{db: db, repos: postgres.NewRepositorySet(db)}
	f.manager = postgres.NewUnitOfWorkManager(db)
	f.wali = leavingSettlementUser(t, domain.RoleWaliKelas)
	f.priorYear = leavingSettlementYearFor(t, 2025)
	f.priorYear.Status = domain.AcademicYearClosed
	f.year = leavingSettlementYear(t)
	f.class = leavingSettlementClass(t, f.year.ID)
	f.student = leavingSettlementStudent(t)
	f.account, err = domain.NewSavingsAccount(f.student.ID, f.year.ID)
	if err != nil {
		t.Fatal(err)
	}
	f.request, err = domain.NewStudentLeavingRequest(
		f.student.ID,
		f.year.ID,
		time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
		f.wali.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	f.assignment = leavingSettlementAssignment(t, f.wali.ID, f.year.ID, f.class.ID)
	f.history = leavingSettlementHistory(t, f.student.ID, f.year.ID, f.class.ID)
	for _, create := range []func() error{
		func() error { return f.repos.Users.Create(ctx, f.wali) },
		func() error { return f.repos.AcademicYears.Create(ctx, f.priorYear) },
		func() error { return f.repos.AcademicYears.Create(ctx, f.year) },
		func() error { return f.repos.Classes.Create(ctx, f.class) },
		func() error { return f.repos.Students.Create(ctx, f.student) },
		func() error { return f.repos.TeacherClassAssignments.Create(ctx, f.assignment) },
		func() error { return f.repos.StudentClassHistories.Create(ctx, f.history) },
		func() error { return f.repos.SavingsAccounts.Create(ctx, f.account) },
		func() error { return f.repos.StudentLeavingRequests.Create(ctx, f.request) },
	} {
		if err := create(); err != nil {
			t.Fatalf("buat fixture: %v", err)
		}
	}
	f.service, err = NewSettlementService(Dependencies{Repositories: f.repos, UOW: f.manager})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.cleanup(t) })
	return f
}

func (f *leavingSettlementFixture) cleanup(t *testing.T) {
	ctx := context.Background()
	statements := []struct {
		query string
		args  []any
	}{
		{`DELETE FROM audit_logs WHERE actor_user_id IN (SELECT id FROM users WHERE username LIKE 'mq08-settle-%') OR entity_id IN (SELECT id FROM students WHERE name LIKE 'Siswa tes settlement %') OR entity_id IN (SELECT id FROM savings_accounts WHERE student_id IN (SELECT id FROM students WHERE name LIKE 'Siswa tes settlement %')) OR entity_id IN (SELECT id FROM student_leaving_requests WHERE student_id IN (SELECT id FROM students WHERE name LIKE 'Siswa tes settlement %'))`, nil},
		{`DELETE FROM settlement_cash_movements WHERE savings_settlement_id IN (SELECT id FROM savings_settlements WHERE savings_account_id IN (SELECT id FROM savings_accounts WHERE student_id IN (SELECT id FROM students WHERE name LIKE 'Siswa tes settlement %')))`, nil},
		{`DELETE FROM settlement_reopen_requests WHERE savings_account_id IN (SELECT id FROM savings_accounts WHERE student_id IN (SELECT id FROM students WHERE name LIKE 'Siswa tes settlement %'))`, nil},
		{`DELETE FROM savings_settlements WHERE savings_account_id IN (SELECT id FROM savings_accounts WHERE student_id IN (SELECT id FROM students WHERE name LIKE 'Siswa tes settlement %'))`, nil},
		{`DELETE FROM leaving_transaction_requests WHERE savings_account_id IN (SELECT id FROM savings_accounts WHERE student_id IN (SELECT id FROM students WHERE name LIKE 'Siswa tes settlement %'))`, nil},
		{`DELETE FROM transactions WHERE savings_account_id IN (SELECT id FROM savings_accounts WHERE student_id IN (SELECT id FROM students WHERE name LIKE 'Siswa tes settlement %'))`, nil},
		{`DELETE FROM savings_accounts WHERE student_id IN (SELECT id FROM students WHERE name LIKE 'Siswa tes settlement %')`, nil},
		{`DELETE FROM student_leaving_requests WHERE student_id IN (SELECT id FROM students WHERE name LIKE 'Siswa tes settlement %')`, nil},
		{`DELETE FROM student_class_histories WHERE student_id IN (SELECT id FROM students WHERE name LIKE 'Siswa tes settlement %')`, nil},
		{`DELETE FROM teacher_class_assignments WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'mq08-settle-%')`, nil},
		{`DELETE FROM students WHERE name LIKE 'Siswa tes settlement %'`, nil},
		{`DELETE FROM classes WHERE name LIKE 'MQ08 settlement %'`, nil},
		{`DELETE FROM academic_years WHERE name LIKE 'MQ08 settlement %'`, nil},
		{`DELETE FROM users WHERE username LIKE 'mq08-settle-%'`, nil},
	}
	for _, statement := range statements {
		if _, err := f.db.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Errorf("bersihkan fixture: %v", err)
		}
	}
}

func (f *leavingSettlementFixture) settle(ctx context.Context, requestID uuid.UUID, accountID uuid.UUID, actor Actor, expected domain.Money) (SettleStudentLeavingOutput, error) {
	return f.service.settleForStudentLeavingRequest(ctx, settleForStudentLeavingRequestInput{
		Actor: actor, StudentLeavingRequestID: requestID, SavingsAccountID: accountID, ExpectedBalance: expected,
	})
}

func (f *leavingSettlementFixture) waliActor() Actor {
	return Actor{UserID: f.wali.ID, Role: domain.RoleWaliKelas}
}

type leavingSettlementTestUOWManager struct {
	base        repository.UnitOfWorkManager
	wrap        func(repository.RepositorySet) repository.RepositorySet
	commitError error
}

func (m leavingSettlementTestUOWManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	uow, err := m.base.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := uow.Repositories()
	if m.wrap != nil {
		repos = m.wrap(repos)
	}
	return leavingSettlementTestUOW{UnitOfWork: uow, repos: repos, commitError: m.commitError}, nil
}

type leavingSettlementTestUOW struct {
	repository.UnitOfWork
	repos       repository.RepositorySet
	commitError error
}

func (u leavingSettlementTestUOW) Repositories() repository.RepositorySet { return u.repos }

func (u leavingSettlementTestUOW) Commit() error {
	if u.commitError != nil {
		return u.commitError
	}
	return u.UnitOfWork.Commit()
}

type leavingSettlementAccountHook struct {
	repository.SavingsAccountRepository
	accountID uuid.UUID
	hook      func(domain.SavingsAccount)
}

func (r leavingSettlementAccountHook) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.SavingsAccount, error) {
	account, err := r.SavingsAccountRepository.GetByIDForUpdate(ctx, id)
	if err == nil && id == r.accountID && r.hook != nil {
		r.hook(account)
	}
	return account, err
}

type leavingSettlementAccountFailure struct {
	repository.SavingsAccountRepository
	failAfterUpdate error
}

func (r leavingSettlementAccountFailure) Update(ctx context.Context, account domain.SavingsAccount) error {
	if err := r.SavingsAccountRepository.Update(ctx, account); err != nil {
		return err
	}
	return r.failAfterUpdate
}

type leavingSettlementCreateFailure struct {
	repository.SettlementRepository
	failAfterInsert error
}

func (r leavingSettlementCreateFailure) CreateForStudentLeavingRequest(ctx context.Context, settlement domain.SavingsSettlement, requestID uuid.UUID) error {
	if err := r.SettlementRepository.CreateForStudentLeavingRequest(ctx, settlement, requestID); err != nil {
		return err
	}
	return r.failAfterInsert
}

type leavingSettlementAuditFailure struct {
	repository.AuditRepository
	failure error
}

func (r leavingSettlementAuditFailure) Create(ctx context.Context, log domain.AuditLog) error {
	return r.failure
}

func (f *leavingSettlementFixture) serviceWithManager(t *testing.T, manager repository.UnitOfWorkManager) *settlementService {
	t.Helper()
	svc, err := NewSettlementService(Dependencies{Repositories: f.repos, UOW: manager})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func waitForLeavingSettlementBlock(t *testing.T, db *sql.DB) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		var blocked bool
		err := db.QueryRowContext(context.Background(), `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE datname = current_database()
				  AND wait_event_type = 'Lock'
				  AND cardinality(pg_blocking_pids(pid)) > 0
			)`).Scan(&blocked)
		if err != nil {
			t.Fatalf("amati pg_blocking_pids: %v", err)
		}
		if blocked {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("tidak melihat transaksi menunggu lock lewat pg_blocking_pids")
}

func leavingSettlementAccountBarrier(accountID uuid.UUID, acquired chan<- struct{}, release <-chan struct{}) func(domain.SavingsAccount) {
	var once sync.Once
	return func(domain.SavingsAccount) {
		once.Do(func() {
			close(acquired)
			<-release
		})
	}
}

func (f *leavingSettlementFixture) addDeposit(amount domain.Money) error {
	tx, err := domain.NewDeposit(f.account.ID, amount, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), f.wali.ID)
	if err != nil {
		return err
	}
	return f.repos.Transactions.Create(context.Background(), tx)
}

func TestLeavingRequestSettlementFunctional(t *testing.T) {
	ctx := context.Background()
	t.Run("saldo positif mengaitkan settlement dan audit", func(t *testing.T) {
		f := newLeavingSettlementFixture(t)
		historicalID := f.createHistoricalSettlementWithoutProcess(t)
		if err := f.addDeposit(60000); err != nil {
			t.Fatal(err)
		}
		out, err := f.settle(ctx, f.request.ID, f.account.ID, f.waliActor(), 60000)
		if err != nil {
			t.Fatal(err)
		}
		assertLeavingSettlementState(t, f, out.Settlement.ID, 60000)
		var linkedID uuid.UUID
		if err := f.db.QueryRowContext(ctx, `SELECT student_leaving_request_id FROM savings_settlements WHERE id=$1`, out.Settlement.ID).Scan(&linkedID); err != nil || linkedID != f.request.ID {
			t.Fatalf("relasi settlement=%s err=%v, ingin %s", linkedID, err, f.request.ID)
		}
		var auditCount int
		var auditPayload []byte
		if err := f.db.QueryRowContext(ctx, `SELECT count(*), (array_agg(after_data))[1] FROM audit_logs WHERE entity_type='SAVINGS_SETTLEMENT' AND entity_id=$1`, out.Settlement.ID).Scan(&auditCount, &auditPayload); err != nil {
			t.Fatal(err)
		}
		var payload struct {
			StudentLeavingRequestID uuid.UUID `json:"student_leaving_request_id"`
		}
		if err := json.Unmarshal(auditPayload, &payload); err != nil || auditCount != 1 || payload.StudentLeavingRequestID != f.request.ID {
			t.Fatalf("audit count=%d payload=%s err=%v", auditCount, auditPayload, err)
		}
		var historicalLink uuid.NullUUID
		var historicalStatus string
		if err := f.db.QueryRowContext(ctx, `SELECT student_leaving_request_id, status FROM savings_settlements WHERE id=$1`, historicalID).Scan(&historicalLink, &historicalStatus); err != nil {
			t.Fatal(err)
		}
		if historicalLink.Valid || historicalStatus != string(domain.SettlementCompleted) {
			t.Fatalf("settlement historis berubah: link=%+v status=%s", historicalLink, historicalStatus)
		}
	})

	t.Run("saldo nol tidak membuat penyerahan", func(t *testing.T) {
		f := newLeavingSettlementFixture(t)
		out, err := f.settle(ctx, f.request.ID, f.account.ID, f.waliActor(), 0)
		if err != nil {
			t.Fatal(err)
		}
		assertLeavingSettlementState(t, f, out.Settlement.ID, 0)
		var movements int
		if err := f.db.QueryRowContext(ctx, `SELECT count(*) FROM settlement_cash_movements WHERE savings_settlement_id=$1`, out.Settlement.ID).Scan(&movements); err != nil || movements != 0 {
			t.Fatalf("cash movements=%d err=%v, want 0", movements, err)
		}
	})

	t.Run("rekening lintas tahun tidak disamakan dengan tahun proses", func(t *testing.T) {
		f := newLeavingSettlementFixture(t)
		f.request.AcademicYearID = f.priorYear.ID
		f.request.EffectiveLeavingDate = time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
		if _, err := f.db.ExecContext(ctx, `UPDATE student_leaving_requests SET academic_year_id=$1, effective_leaving_date=$2 WHERE id=$3`, f.priorYear.ID, f.request.EffectiveLeavingDate, f.request.ID); err != nil {
			t.Fatal(err)
		}
		out, err := f.settle(ctx, f.request.ID, f.account.ID, f.waliActor(), 0)
		if err != nil || out.Settlement == nil {
			t.Fatalf("settle across reference year=%+v err=%v", out, err)
		}
	})

	tests := []struct {
		name    string
		prepare func(*testing.T, *leavingSettlementFixture) (uuid.UUID, uuid.UUID, Actor, domain.Money)
		want    error
	}{
		{name: "proses tidak ditemukan", prepare: func(_ *testing.T, f *leavingSettlementFixture) (uuid.UUID, uuid.UUID, Actor, domain.Money) {
			return uuid.New(), f.account.ID, f.waliActor(), 0
		}, want: repository.ErrNotFound},
		{name: "proses dibatalkan", prepare: func(t *testing.T, f *leavingSettlementFixture) (uuid.UUID, uuid.UUID, Actor, domain.Money) {
			if err := f.cancelRequest(); err != nil {
				t.Fatal(err)
			}
			return f.request.ID, f.account.ID, f.waliActor(), 0
		}, want: domain.ErrInvalidTransition},
		{name: "proses difinalisasi", prepare: func(t *testing.T, f *leavingSettlementFixture) (uuid.UUID, uuid.UUID, Actor, domain.Money) {
			if err := f.finalizeRequest(); err != nil {
				t.Fatal(err)
			}
			return f.request.ID, f.account.ID, f.waliActor(), 0
		}, want: domain.ErrInvalidTransition},
		{name: "rekening siswa lain", prepare: func(t *testing.T, f *leavingSettlementFixture) (uuid.UUID, uuid.UUID, Actor, domain.Money) {
			other := leavingSettlementStudent(t)
			if err := f.repos.Students.Create(ctx, other); err != nil {
				t.Fatal(err)
			}
			second, err := domain.NewSavingsAccount(other.ID, f.year.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.repos.SavingsAccounts.Create(ctx, second); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_, _ = f.db.ExecContext(ctx, `DELETE FROM savings_accounts WHERE id=$1`, second.ID)
				_, _ = f.db.ExecContext(ctx, `DELETE FROM students WHERE id=$1`, other.ID)
			})
			return f.request.ID, second.ID, f.waliActor(), 0
		}, want: domain.ErrInvalidTransition},
		{name: "siswa tidak aktif", prepare: func(t *testing.T, f *leavingSettlementFixture) (uuid.UUID, uuid.UUID, Actor, domain.Money) {
			if err := f.student.LeaveOn(time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)); err != nil {
				t.Fatal(err)
			}
			if err := f.repos.Students.Update(ctx, f.student); err != nil {
				t.Fatal(err)
			}
			return f.request.ID, f.account.ID, f.waliActor(), 0
		}, want: domain.ErrInvalidTransition},
		{name: "rekening sudah selesai", prepare: func(t *testing.T, f *leavingSettlementFixture) (uuid.UUID, uuid.UUID, Actor, domain.Money) {
			if err := f.account.Settle(time.Now()); err != nil {
				t.Fatal(err)
			}
			if err := f.repos.SavingsAccounts.Update(ctx, f.account); err != nil {
				t.Fatal(err)
			}
			old, err := domain.NewSavingsSettlement(f.account.ID, domain.SettlementStudentLeaving, 0, f.wali.ID, time.Now(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.repos.Settlements.Create(ctx, old); err != nil {
				t.Fatal(err)
			}
			return f.request.ID, f.account.ID, f.waliActor(), 0
		}, want: domain.ErrInvalidTransition},
		{name: "tahun rekening tertutup", prepare: func(t *testing.T, f *leavingSettlementFixture) (uuid.UUID, uuid.UUID, Actor, domain.Money) {
			if err := f.account.Settle(time.Now()); err != nil {
				t.Fatal(err)
			}
			if err := f.repos.SavingsAccounts.Update(ctx, f.account); err != nil {
				t.Fatal(err)
			}
			old, err := domain.NewSavingsSettlement(f.account.ID, domain.SettlementStudentLeaving, 0, f.wali.ID, time.Now(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.repos.Settlements.Create(ctx, old); err != nil {
				t.Fatal(err)
			}
			if _, err := f.db.ExecContext(ctx, `UPDATE academic_years SET status='CLOSED' WHERE id=$1`, f.year.ID); err != nil {
				t.Fatal(err)
			}
			return f.request.ID, f.account.ID, f.waliActor(), 0
		}, want: domain.ErrInvalidTransition},
		{name: "wali di luar scope", prepare: func(t *testing.T, f *leavingSettlementFixture) (uuid.UUID, uuid.UUID, Actor, domain.Money) {
			other := leavingSettlementUser(t, domain.RoleWaliKelas)
			if err := f.repos.Users.Create(ctx, other); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _, _ = f.db.ExecContext(ctx, `DELETE FROM users WHERE id=$1`, other.ID) })
			return f.request.ID, f.account.ID, Actor{UserID: other.ID, Role: domain.RoleWaliKelas}, 0
		}, want: ErrScopeViolation},
		{name: "admin tidak boleh melakukan settlement", prepare: func(t *testing.T, f *leavingSettlementFixture) (uuid.UUID, uuid.UUID, Actor, domain.Money) {
			admin := leavingSettlementUser(t, domain.RoleAdmin)
			if err := f.repos.Users.Create(ctx, admin); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _, _ = f.db.ExecContext(ctx, `DELETE FROM users WHERE id=$1`, admin.ID) })
			return f.request.ID, f.account.ID, Actor{UserID: admin.ID, Role: domain.RoleAdmin}, 0
		}, want: ErrForbidden},
		{name: "saldo stale", prepare: func(t *testing.T, f *leavingSettlementFixture) (uuid.UUID, uuid.UUID, Actor, domain.Money) {
			if err := f.addDeposit(100); err != nil {
				t.Fatal(err)
			}
			return f.request.ID, f.account.ID, f.waliActor(), 0
		}, want: repository.ErrConflict},
		{name: "late request pending", prepare: func(t *testing.T, f *leavingSettlementFixture) (uuid.UUID, uuid.UUID, Actor, domain.Money) {
			if err := f.addLateRequest(domain.LeavingTransactionPending); err != nil {
				t.Fatal(err)
			}
			return f.request.ID, f.account.ID, f.waliActor(), 0
		}, want: domain.ErrInvalidTransition},
		{name: "late request approved", prepare: func(t *testing.T, f *leavingSettlementFixture) (uuid.UUID, uuid.UUID, Actor, domain.Money) {
			if err := f.addLateRequest(domain.LeavingTransactionApproved); err != nil {
				t.Fatal(err)
			}
			return f.request.ID, f.account.ID, f.waliActor(), 0
		}, want: domain.ErrInvalidTransition},
		{name: "settlement terdahulu tidak diganti", prepare: func(t *testing.T, f *leavingSettlementFixture) (uuid.UUID, uuid.UUID, Actor, domain.Money) {
			old, err := domain.NewSavingsSettlement(f.account.ID, domain.SettlementStudentLeaving, 0, f.wali.ID, time.Now(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := old.Supersede(); err != nil {
				t.Fatal(err)
			}
			if err := f.repos.Settlements.Create(ctx, old); err != nil {
				t.Fatal(err)
			}
			return f.request.ID, f.account.ID, f.waliActor(), 0
		}, want: domain.ErrInvalidTransition},
		{name: "relasi historis tidak konsisten diblokir", prepare: func(t *testing.T, f *leavingSettlementFixture) (uuid.UUID, uuid.UUID, Actor, domain.Money) {
			_, err := f.db.ExecContext(ctx, `INSERT INTO savings_settlements (id,savings_account_id,type,amount,executed_by,executed_at,status,student_leaving_request_id) VALUES ($1,$2,'REGULAR_YEAR_END',0,$3,now(),'SUPERSEDED',$4)`, uuid.New(), f.account.ID, f.wali.ID, f.request.ID)
			if err != nil {
				t.Fatal(err)
			}
			return f.request.ID, f.account.ID, f.waliActor(), 0
		}, want: domain.ErrInvalidTransition},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newLeavingSettlementFixture(t)
			requestID, accountID, actor, expected := test.prepare(t, f)
			before := f.snapshot(t)
			_, err := f.settle(ctx, requestID, accountID, actor, expected)
			if !errors.Is(err, test.want) {
				t.Fatalf("error=%v, ingin %v", err, test.want)
			}
			f.assertSnapshot(t, before)
		})
	}
}

func (f *leavingSettlementFixture) createHistoricalSettlementWithoutProcess(t *testing.T) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	student := leavingSettlementStudent(t)
	if err := f.repos.Students.Create(ctx, student); err != nil {
		t.Fatal(err)
	}
	account, err := domain.NewSavingsAccount(student.ID, f.year.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.SavingsAccounts.Create(ctx, account); err != nil {
		t.Fatal(err)
	}
	if err := account.Settle(time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := f.repos.SavingsAccounts.Update(ctx, account); err != nil {
		t.Fatal(err)
	}
	settlement, err := domain.NewSavingsSettlement(account.ID, domain.SettlementStudentLeaving, 0, f.wali.ID, time.Now(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.Settlements.Create(ctx, settlement); err != nil {
		t.Fatal(err)
	}
	return settlement.ID
}

func (f *leavingSettlementFixture) addLateRequest(status domain.LeavingTransactionRequestStatus) error {
	admin := leavingSettlementUserForSetup(domain.RoleAdmin)
	if err := f.repos.Users.Create(context.Background(), admin); err != nil {
		return err
	}
	request, err := domain.NewLeavingTransactionRequest(f.account.ID, f.year.ID, domain.TransactionDeposit, 10, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), "transaksi terlambat", "bukti-late-"+uuid.NewString(), f.wali.ID)
	if err != nil {
		return err
	}
	if status == domain.LeavingTransactionApproved {
		if err := request.Decide(admin.ID, true, "disetujui untuk tes"); err != nil {
			return err
		}
	}
	return f.repos.LeavingTransactionRequests.CreateLeavingTransactionRequest(context.Background(), request)
}

func TestLeavingRequestSettlementConcurrency(t *testing.T) {
	t.Run("settlement menserialisasi setoran dan penarikan", func(t *testing.T) {
		for _, kind := range []string{"setoran", "penarikan"} {
			t.Run(kind, func(t *testing.T) {
				f := newLeavingSettlementFixture(t)
				if err := f.addDeposit(1000); err != nil {
					t.Fatal(err)
				}
				acquired, release := make(chan struct{}), make(chan struct{})
				hook := leavingSettlementAccountBarrier(f.account.ID, acquired, release)
				manager := leavingSettlementTestUOWManager{base: f.manager, wrap: func(repos repository.RepositorySet) repository.RepositorySet {
					repos.SavingsAccounts = leavingSettlementAccountHook{SavingsAccountRepository: repos.SavingsAccounts, accountID: f.account.ID, hook: hook}
					return repos
				}}
				svc := f.serviceWithManager(t, manager)
				type result struct {
					out SettleStudentLeavingOutput
					err error
				}
				settled := make(chan result, 1)
				go func() {
					out, err := svc.settleForStudentLeavingRequest(context.Background(), settleForStudentLeavingRequestInput{Actor: f.waliActor(), StudentLeavingRequestID: f.request.ID, SavingsAccountID: f.account.ID, ExpectedBalance: 1000})
					settled <- result{out, err}
				}()
				waitForSignal(t, acquired, "settlement mengunci rekening")
				transactionDone := make(chan error, 1)
				go func() {
					txSvc, err := NewTransactionService(Dependencies{Repositories: f.repos, UOW: f.manager})
					if err != nil {
						transactionDone <- err
						return
					}
					date := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
					if kind == "setoran" {
						_, err = txSvc.Deposit(context.Background(), DepositInput{Actor: f.waliActor(), SavingsAccountID: f.account.ID, Amount: 100, TransactionDate: date})
					} else {
						_, err = txSvc.Withdrawal(context.Background(), WithdrawalInput{Actor: f.waliActor(), SavingsAccountID: f.account.ID, Amount: 100, TransactionDate: date})
					}
					transactionDone <- err
				}()
				waitForLeavingSettlementBlock(t, f.db)
				close(release)
				settlementResult := awaitLeavingSettlement(t, settled)
				if settlementResult.err != nil {
					t.Fatalf("settlement: %v", settlementResult.err)
				}
				if err := awaitLeavingSettlementError(t, transactionDone); !errors.Is(err, domain.ErrInvalidTransition) {
					t.Fatalf("transaksi setelah settlement=%v", err)
				}
				assertLeavingSettlementState(t, f, settlementResult.out.Settlement.ID, 1000)
				var auditCount int
				if err := f.db.QueryRowContext(context.Background(), `SELECT count(*) FROM audit_logs WHERE actor_user_id=$1`, f.wali.ID).Scan(&auditCount); err != nil || auditCount != 2 {
					t.Fatalf("audit settlement=%d err=%v, ingin 2", auditCount, err)
				}
				var txCount int
				if err := f.db.QueryRowContext(context.Background(), `SELECT count(*) FROM transactions WHERE savings_account_id=$1`, f.account.ID).Scan(&txCount); err != nil || txCount != 1 {
					t.Fatalf("jumlah transaksi=%d err=%v, ingin 1", txCount, err)
				}
			})
		}
	})

	t.Run("setoran lebih dahulu membuat preview settlement stale", func(t *testing.T) {
		f := newLeavingSettlementFixture(t)
		if err := f.addDeposit(1000); err != nil {
			t.Fatal(err)
		}
		acquired, release := make(chan struct{}), make(chan struct{})
		hook := leavingSettlementAccountBarrier(f.account.ID, acquired, release)
		manager := leavingSettlementTestUOWManager{base: f.manager, wrap: func(repos repository.RepositorySet) repository.RepositorySet {
			repos.SavingsAccounts = leavingSettlementAccountHook{SavingsAccountRepository: repos.SavingsAccounts, accountID: f.account.ID, hook: hook}
			return repos
		}}
		txSvc, err := NewTransactionService(Dependencies{Repositories: f.repos, UOW: manager})
		if err != nil {
			t.Fatal(err)
		}
		depositDone := make(chan error, 1)
		go func() {
			_, err := txSvc.Deposit(context.Background(), DepositInput{Actor: f.waliActor(), SavingsAccountID: f.account.ID, Amount: 100, TransactionDate: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)})
			depositDone <- err
		}()
		waitForSignal(t, acquired, "setoran mengunci rekening")
		settlementDone := make(chan error, 1)
		go func() {
			_, err := f.settle(context.Background(), f.request.ID, f.account.ID, f.waliActor(), 1000)
			settlementDone <- err
		}()
		waitForLeavingSettlementBlock(t, f.db)
		close(release)
		if err := awaitLeavingSettlementError(t, depositDone); err != nil {
			t.Fatalf("setoran: %v", err)
		}
		if err := awaitLeavingSettlementError(t, settlementDone); !errors.Is(err, repository.ErrConflict) {
			t.Fatalf("settlement dengan saldo lama=%v, ingin conflict", err)
		}
		var status string
		var transactionCount, settlementCount int
		if err := f.db.QueryRowContext(context.Background(), `SELECT status FROM savings_accounts WHERE id=$1`, f.account.ID).Scan(&status); err != nil {
			t.Fatal(err)
		}
		if err := f.db.QueryRowContext(context.Background(), `SELECT count(*) FROM transactions WHERE savings_account_id=$1`, f.account.ID).Scan(&transactionCount); err != nil {
			t.Fatal(err)
		}
		if err := f.db.QueryRowContext(context.Background(), `SELECT count(*) FROM savings_settlements WHERE savings_account_id=$1`, f.account.ID).Scan(&settlementCount); err != nil {
			t.Fatal(err)
		}
		if status != string(domain.SavingsAccountOpen) || transactionCount != 2 || settlementCount != 0 {
			t.Fatalf("hasil partial status=%s tx=%d settlements=%d", status, transactionCount, settlementCount)
		}
		var auditCount int
		if err := f.db.QueryRowContext(context.Background(), `SELECT count(*) FROM audit_logs WHERE actor_user_id=$1`, f.wali.ID).Scan(&auditCount); err != nil || auditCount != 1 {
			t.Fatalf("audit setelah setoran=%d err=%v, ingin 1", auditCount, err)
		}
	})

	t.Run("dua settlement rekening yang sama hanya satu berhasil", func(t *testing.T) {
		f := newLeavingSettlementFixture(t)
		acquired, release := make(chan struct{}), make(chan struct{})
		hook := leavingSettlementAccountBarrier(f.account.ID, acquired, release)
		manager := leavingSettlementTestUOWManager{base: f.manager, wrap: func(repos repository.RepositorySet) repository.RepositorySet {
			repos.SavingsAccounts = leavingSettlementAccountHook{SavingsAccountRepository: repos.SavingsAccounts, accountID: f.account.ID, hook: hook}
			return repos
		}}
		firstService := f.serviceWithManager(t, manager)
		type result struct {
			out SettleStudentLeavingOutput
			err error
		}
		firstDone, secondDone := make(chan result, 1), make(chan result, 1)
		call := func(svc *settlementService, done chan<- result) {
			out, err := svc.settleForStudentLeavingRequest(context.Background(), settleForStudentLeavingRequestInput{Actor: f.waliActor(), StudentLeavingRequestID: f.request.ID, SavingsAccountID: f.account.ID, ExpectedBalance: 0})
			done <- result{out, err}
		}
		go call(firstService, firstDone)
		waitForSignal(t, acquired, "settlement pertama mengunci rekening")
		go call(f.service, secondDone)
		waitForLeavingSettlementBlock(t, f.db)
		close(release)
		first, second := awaitLeavingSettlement(t, firstDone), awaitLeavingSettlement(t, secondDone)
		if first.err != nil || !errors.Is(second.err, domain.ErrInvalidTransition) {
			t.Fatalf("hasil settlement bersamaan pertama=%v kedua=%v", first.err, second.err)
		}
		var count, audits int
		if err := f.db.QueryRowContext(context.Background(), `SELECT count(*) FROM savings_settlements WHERE savings_account_id=$1`, f.account.ID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if err := f.db.QueryRowContext(context.Background(), `SELECT count(*) FROM audit_logs WHERE entity_id=$1 OR entity_id=$2 OR entity_id=$3`, f.account.ID, f.student.ID, first.out.Settlement.ID).Scan(&audits); err != nil {
			t.Fatal(err)
		}
		if count != 1 || audits != 2 {
			t.Fatalf("settlements=%d audits=%d, ingin 1/2", count, audits)
		}
	})

	t.Run("perubahan status siswa lebih dahulu dibaca ulang", func(t *testing.T) {
		f := newLeavingSettlementFixture(t)
		blocker, err := f.db.BeginTx(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer blocker.Rollback()
		if _, err := blocker.ExecContext(context.Background(), `SELECT id FROM students WHERE id=$1 FOR UPDATE`, f.student.ID); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() {
			_, err := f.settle(context.Background(), f.request.ID, f.account.ID, f.waliActor(), 0)
			done <- err
		}()
		waitForLeavingSettlementBlock(t, f.db)
		if _, err := blocker.ExecContext(context.Background(), `UPDATE students SET status='LEFT', leaving_date=$2 WHERE id=$1`, f.student.ID, time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}
		if err := blocker.Commit(); err != nil {
			t.Fatal(err)
		}
		if err := awaitLeavingSettlementError(t, done); !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("settlement setelah siswa LEFT=%v", err)
		}
		f.assertUnchanged(t)
	})

	t.Run("penutupan tahun lebih dahulu memblokir settlement", func(t *testing.T) {
		f := newLeavingSettlementFixture(t)
		blocker, err := f.db.BeginTx(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer blocker.Rollback()
		if _, err := postgres.NewAcademicYearRepository(blocker).GetByIDForUpdate(context.Background(), f.year.ID); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() {
			_, err := f.settle(context.Background(), f.request.ID, f.account.ID, f.waliActor(), 0)
			done <- err
		}()
		waitForLeavingSettlementBlock(t, f.db)
		// Trigger penutupan menolak rekening OPEN; fixture sementara meniru data legacy
		// yang harus tetap diblokir settlement tanpa mengubah definisi trigger.
		if _, err := blocker.ExecContext(context.Background(), `SET LOCAL session_replication_role = replica`); err != nil {
			t.Fatal(err)
		}
		if _, err := blocker.ExecContext(context.Background(), `UPDATE academic_years SET status='CLOSED' WHERE id=$1`, f.year.ID); err != nil {
			t.Fatal(err)
		}
		if err := blocker.Commit(); err != nil {
			t.Fatal(err)
		}
		if err := awaitLeavingSettlementError(t, done); !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("settlement setelah tahun CLOSED=%v", err)
		}
		f.assertUnchanged(t)
	})

	t.Run("pembatalan proses lebih dahulu dibaca ulang", func(t *testing.T) {
		f := newLeavingSettlementFixture(t)
		blocker, err := f.db.BeginTx(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer blocker.Rollback()
		repos := postgres.NewRepositorySet(blocker)
		if _, err := repos.AcademicYears.GetByIDForShare(context.Background(), f.year.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := repos.SavingsAccounts.GetByIDForUpdate(context.Background(), f.account.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := repos.Students.GetByIDForUpdate(context.Background(), f.student.ID); err != nil {
			t.Fatal(err)
		}
		request, err := repos.StudentLeavingRequests.GetByIDForUpdate(context.Background(), f.request.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := request.Cancel(f.wali.ID, time.Now(), "dibatalkan dalam tes konkurensi"); err != nil {
			t.Fatal(err)
		}
		if err := repos.StudentLeavingRequests.Update(context.Background(), request); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() {
			_, err := f.settle(context.Background(), f.request.ID, f.account.ID, f.waliActor(), 0)
			done <- err
		}()
		waitForLeavingSettlementBlock(t, f.db)
		if err := blocker.Commit(); err != nil {
			t.Fatal(err)
		}
		if err := awaitLeavingSettlementError(t, done); !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("settlement setelah CANCELLED=%v", err)
		}
		var requestStatus string
		if err := f.db.QueryRowContext(context.Background(), `SELECT status FROM student_leaving_requests WHERE id=$1`, f.request.ID).Scan(&requestStatus); err != nil || requestStatus != string(domain.StudentLeavingCancelled) {
			t.Fatalf("status proses setelah interleaving=%s err=%v", requestStatus, err)
		}
		f.assertUnchanged(t)
	})
}

func waitForSignal(t *testing.T, ch <-chan struct{}, label string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(8 * time.Second):
		t.Fatalf("timeout menunggu %s", label)
	}
}

func awaitLeavingSettlement[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case result := <-ch:
		return result
	case <-time.After(10 * time.Second):
		t.Fatal("timeout menunggu transaksi settlement")
		var zero T
		return zero
	}
}

func awaitLeavingSettlementError(t *testing.T, ch <-chan error) error {
	return awaitLeavingSettlement(t, ch)
}

func TestLeavingRequestSettlementRollback(t *testing.T) {
	ctx := context.Background()
	failure := errors.New("kegagalan injeksi tes")
	tests := []struct {
		name        string
		wrap        func(repository.RepositorySet) repository.RepositorySet
		commitError error
	}{
		{name: "setelah perubahan rekening", wrap: func(repos repository.RepositorySet) repository.RepositorySet {
			repos.SavingsAccounts = leavingSettlementAccountFailure{SavingsAccountRepository: repos.SavingsAccounts, failAfterUpdate: failure}
			return repos
		}},
		{name: "setelah insert settlement", wrap: func(repos repository.RepositorySet) repository.RepositorySet {
			repos.Settlements = leavingSettlementCreateFailure{SettlementRepository: repos.Settlements, failAfterInsert: failure}
			return repos
		}},
		{name: "saat menulis audit", wrap: func(repos repository.RepositorySet) repository.RepositorySet {
			repos.AuditLogs = leavingSettlementAuditFailure{AuditRepository: repos.AuditLogs, failure: failure}
			return repos
		}},
		{name: "sebelum commit", commitError: failure},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newLeavingSettlementFixture(t)
			manager := leavingSettlementTestUOWManager{base: f.manager, wrap: test.wrap, commitError: test.commitError}
			svc := f.serviceWithManager(t, manager)
			_, err := svc.settleForStudentLeavingRequest(ctx, settleForStudentLeavingRequestInput{Actor: f.waliActor(), StudentLeavingRequestID: f.request.ID, SavingsAccountID: f.account.ID, ExpectedBalance: 0})
			if !errors.Is(err, failure) {
				t.Fatalf("error=%v, ingin kegagalan injeksi", err)
			}
			f.assertUnchanged(t)
			var processStatus string
			if err := f.db.QueryRowContext(ctx, `SELECT status FROM student_leaving_requests WHERE id=$1`, f.request.ID).Scan(&processStatus); err != nil || processStatus != string(domain.StudentLeavingInProgress) {
				t.Fatalf("status proses=%s err=%v", processStatus, err)
			}
		})
	}
}

func (f *leavingSettlementFixture) cancelRequest() error {
	if err := f.request.Cancel(f.wali.ID, time.Now(), "tes pembatalan"); err != nil {
		return err
	}
	return f.repos.StudentLeavingRequests.Update(context.Background(), f.request)
}

func (f *leavingSettlementFixture) finalizeRequest() error {
	if err := f.request.Finalize(f.wali.ID, time.Now()); err != nil {
		return err
	}
	return f.repos.StudentLeavingRequests.Update(context.Background(), f.request)
}

func (f *leavingSettlementFixture) assertUnchanged(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	var accountStatus string
	var settlementCount, auditCount int
	if err := f.db.QueryRowContext(ctx, `SELECT status FROM savings_accounts WHERE id=$1`, f.account.ID).Scan(&accountStatus); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(ctx, `SELECT count(*) FROM savings_settlements WHERE savings_account_id=$1`, f.account.ID).Scan(&settlementCount); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(ctx, `SELECT count(*) FROM audit_logs WHERE actor_user_id=$1`, f.wali.ID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if accountStatus != string(domain.SavingsAccountOpen) || settlementCount != 0 || auditCount != 0 {
		t.Fatalf("perubahan parsial: account=%s settlements=%d audits=%d", accountStatus, settlementCount, auditCount)
	}
}

type leavingSettlementSnapshot struct {
	accountStatus, studentStatus, processStatus  string
	transactions, settlements, movements, audits int
}

func (f *leavingSettlementFixture) snapshot(t *testing.T) leavingSettlementSnapshot {
	t.Helper()
	ctx := context.Background()
	var snapshot leavingSettlementSnapshot
	queries := []struct {
		query string
		dest  any
		args  []any
	}{
		{`SELECT status FROM savings_accounts WHERE id=$1`, &snapshot.accountStatus, []any{f.account.ID}},
		{`SELECT status FROM students WHERE id=$1`, &snapshot.studentStatus, []any{f.student.ID}},
		{`SELECT status FROM student_leaving_requests WHERE id=$1`, &snapshot.processStatus, []any{f.request.ID}},
		{`SELECT count(*) FROM transactions WHERE savings_account_id=$1`, &snapshot.transactions, []any{f.account.ID}},
		{`SELECT count(*) FROM savings_settlements WHERE savings_account_id=$1`, &snapshot.settlements, []any{f.account.ID}},
		{`SELECT count(*) FROM settlement_cash_movements WHERE savings_settlement_id IN (SELECT id FROM savings_settlements WHERE savings_account_id=$1)`, &snapshot.movements, []any{f.account.ID}},
		{`SELECT count(*) FROM audit_logs WHERE actor_user_id=$1`, &snapshot.audits, []any{f.wali.ID}},
	}
	for _, query := range queries {
		if err := f.db.QueryRowContext(ctx, query.query, query.args...).Scan(query.dest); err != nil {
			t.Fatalf("snapshot query %q: %v", query.query, err)
		}
	}
	return snapshot
}

func (f *leavingSettlementFixture) assertSnapshot(t *testing.T, before leavingSettlementSnapshot) {
	t.Helper()
	after := f.snapshot(t)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("kondisi berubah setelah settlement ditolak: before=%+v after=%+v", before, after)
	}
}

func assertLeavingSettlementState(t *testing.T, f *leavingSettlementFixture, settlementID uuid.UUID, amount domain.Money) {
	t.Helper()
	ctx := context.Background()
	var accountStatus string
	var studentStatus string
	var processStatus string
	var settlementAmount domain.Money
	var settlementType string
	var handovers int
	if err := f.db.QueryRowContext(ctx, `SELECT status FROM savings_accounts WHERE id=$1`, f.account.ID).Scan(&accountStatus); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(ctx, `SELECT status FROM students WHERE id=$1`, f.student.ID).Scan(&studentStatus); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(ctx, `SELECT status FROM student_leaving_requests WHERE id=$1`, f.request.ID).Scan(&processStatus); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(ctx, `SELECT amount::bigint, type FROM savings_settlements WHERE id=$1`, settlementID).Scan(&settlementAmount, &settlementType); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(ctx, `SELECT count(*) FROM settlement_cash_movements WHERE savings_settlement_id=$1`, settlementID).Scan(&handovers); err != nil {
		t.Fatal(err)
	}
	if accountStatus != string(domain.SavingsAccountSettled) || studentStatus != string(domain.StudentActive) || processStatus != string(domain.StudentLeavingInProgress) || settlementAmount != amount || settlementType != string(domain.SettlementStudentLeaving) || handovers != 0 {
		t.Fatalf("status account/student/process=%s/%s/%s settlement=%s/%v handovers=%d", accountStatus, studentStatus, processStatus, settlementType, settlementAmount, handovers)
	}
}

func leavingSettlementUser(t *testing.T, role domain.UserRole) domain.User {
	t.Helper()
	user, err := domain.NewUser("mq08-settle-"+uuid.NewString(), "test-password-hash", role)
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func leavingSettlementUserForSetup(role domain.UserRole) domain.User {
	user, _ := domain.NewUser("mq08-settle-"+uuid.NewString(), "test-password-hash", role)
	return user
}

func leavingSettlementYear(t *testing.T) domain.AcademicYear {
	return leavingSettlementYearFor(t, 2026)
}

func leavingSettlementYearFor(t *testing.T, startYear int) domain.AcademicYear {
	t.Helper()
	year, err := domain.NewAcademicYear("MQ08 settlement "+uuid.NewString(), time.Date(startYear, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(startYear+1, 7, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return year
}

func leavingSettlementClass(t *testing.T, yearID uuid.UUID) domain.Class {
	t.Helper()
	class, err := domain.NewClass("MQ08 settlement "+uuid.NewString(), 8)
	if err != nil {
		t.Fatal(err)
	}
	class.AcademicYearID = yearID
	class.EffectiveDate = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	return class
}

func leavingSettlementStudent(t *testing.T) domain.Student {
	t.Helper()
	nis := "NIS-" + uuid.NewString()
	nisn := "NISN-" + uuid.NewString()
	student, err := domain.NewStudent("Siswa tes settlement "+uuid.NewString(), nil, &nis, &nisn)
	if err != nil {
		t.Fatal(err)
	}
	return student
}

func leavingSettlementAssignment(t *testing.T, userID, yearID, classID uuid.UUID) domain.TeacherClassAssignment {
	t.Helper()
	to := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)
	period, err := domain.NewDateRange(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), &to)
	if err != nil {
		t.Fatal(err)
	}
	assignment, err := domain.NewTeacherClassAssignment(userID, yearID, classID, period)
	if err != nil {
		t.Fatal(err)
	}
	return assignment
}

func leavingSettlementHistory(t *testing.T, studentID, yearID, classID uuid.UUID) domain.StudentClassHistory {
	t.Helper()
	to := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)
	period, err := domain.NewDateRange(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), &to)
	if err != nil {
		t.Fatal(err)
	}
	history, err := domain.NewStudentClassHistory(studentID, yearID, classID, period)
	if err != nil {
		t.Fatal(err)
	}
	return history
}

package service_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/repository/postgres"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type leavingStartFixture struct {
	db      *sql.DB
	repos   repository.RepositorySet
	manager repository.UnitOfWorkManager
	svc     service.StudentLeavingRequestService
	in      service.StartStudentLeavingRequestInput
	year    domain.AcademicYear
	class   domain.Class
}

func newLeavingStartFixture(t *testing.T) leavingStartFixture {
	return newLeavingRequestFixture(t, 15)
}

func newLeavingCancelFixture(t *testing.T) leavingStartFixture {
	return newLeavingRequestFixture(t, 17)
}

func newLeavingRequestFixture(t *testing.T, requiredSchema int) leavingStartFixture {
	t.Helper()
	ctx := context.Background()
	db := openServiceScopeTestDatabase(t)
	var version int
	var dirty bool
	if err := db.QueryRowContext(ctx, `SELECT version,dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty); err != nil {
		t.Fatal(err)
	}
	if version != requiredSchema || dirty {
		t.Skipf("requires clean schema v%d; version=%d dirty=%t", requiredSchema, version, dirty)
	}
	cleanupServiceScopeFixtures(t, db)
	t.Cleanup(func() { cleanupServiceScopeFixtures(t, db) })
	f := leavingStartFixture{db: db, repos: postgres.NewRepositorySet(db), manager: postgres.NewUnitOfWorkManager(db)}
	admin := newServiceScopeUser(t, domain.RoleAdmin)
	if err := f.repos.Users.Create(ctx, admin); err != nil {
		t.Fatal(err)
	}
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().In(location)
	startYear := now.Year()
	if now.Month() < time.July {
		startYear--
	}
	f.year = newServiceScopeAcademicYearForStartYear(t, "Tahun uji proses pindah", startYear)
	if err := f.repos.AcademicYears.Create(ctx, f.year); err != nil {
		t.Fatal(err)
	}
	f.class = newServiceScopeClass(t, "Kelas proses pindah", 7)
	f.class.AcademicYearID = f.year.ID
	f.class.EffectiveDate = f.year.StartDate
	if err := f.repos.Classes.Create(ctx, f.class); err != nil {
		t.Fatal(err)
	}
	student := f.addStudent(t)
	f.in = service.StartStudentLeavingRequestInput{Actor: service.Actor{UserID: admin.ID, Role: domain.RoleAdmin}, StudentID: student.ID, AcademicYearID: f.year.ID, EffectiveLeavingDate: serviceDateOnlyUTC(now)}
	f.svc, err = service.NewStudentLeavingRequestService(service.Dependencies{Repositories: f.repos, UOW: f.manager})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func serviceDateOnlyUTC(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func (f leavingStartFixture) addStudent(t *testing.T) domain.Student {
	t.Helper()
	ctx := context.Background()
	student := newServiceScopeStudent(t, "Siswa fixture proses pindah")
	// Riwayat tanggal pindah lama dapat tetap ada setelah reaktivasi; Start tidak boleh menulisnya ulang.
	oldDate := f.year.StartDate.AddDate(-1, 0, 0)
	student.LeavingDate = &oldDate
	if err := f.repos.Students.Create(ctx, student); err != nil {
		t.Fatal(err)
	}
	period, err := domain.NewDateRange(f.year.StartDate, &f.year.EndDate)
	if err != nil {
		t.Fatal(err)
	}
	history, err := domain.NewStudentClassHistory(student.ID, f.year.ID, f.class.ID, period)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.StudentClassHistories.Create(ctx, history); err != nil {
		t.Fatal(err)
	}
	return student
}

func (f leavingStartFixture) preservedSnapshot(t *testing.T) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	for _, table := range []string{"students", "academic_years", "student_class_histories", "savings_accounts", "transactions", "savings_settlements", "settlement_cash_movements"} {
		var value string
		if err := f.db.QueryRow(`SELECT COALESCE(jsonb_agg(to_jsonb(t) ORDER BY id),'[]'::jsonb)::text FROM ` + table + ` t`).Scan(&value); err != nil {
			t.Fatal(err)
		}
		snapshot[table] = value
	}
	return snapshot
}
func (f leavingStartFixture) assertCounts(t *testing.T, requests, audits int) {
	t.Helper()
	var gotRequests, gotAudits int
	if err := f.db.QueryRow(`SELECT (SELECT count(*) FROM student_leaving_requests),(SELECT count(*) FROM audit_logs WHERE action='STUDENT_LEAVING_STARTED')`).Scan(&gotRequests, &gotAudits); err != nil {
		t.Fatal(err)
	}
	if gotRequests != requests || gotAudits != audits {
		t.Fatalf("requests/audits=%d/%d, want %d/%d", gotRequests, gotAudits, requests, audits)
	}
}
func (f leavingStartFixture) assertAudit(t *testing.T, request domain.StudentLeavingRequest) {
	t.Helper()
	logs, err := f.repos.AuditLogs.ListByEntity(context.Background(), domain.AuditEntityStudent, request.StudentID, repository.ListOptions{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	var found int
	for _, a := range logs {
		if a.Action != "STUDENT_LEAVING_STARTED" {
			continue
		}
		var p struct {
			StudentID uuid.UUID                          `json:"student_id"`
			RequestID uuid.UUID                          `json:"leaving_request_id"`
			YearID    uuid.UUID                          `json:"academic_year_id"`
			Date      string                             `json:"effective_leaving_date"`
			Status    domain.StudentLeavingRequestStatus `json:"status"`
			ActorID   uuid.UUID                          `json:"actor_id"`
			ActorRole domain.UserRole                    `json:"actor_role"`
			At        time.Time                          `json:"occurred_at"`
		}
		if err := json.Unmarshal(a.AfterData, &p); err != nil {
			t.Fatal(err)
		}
		if p.RequestID != request.ID {
			continue
		}
		found++
		if a.ActorUserID == nil || *a.ActorUserID != f.in.Actor.UserID || a.BeforeData != nil || p.StudentID != request.StudentID || p.YearID != request.AcademicYearID || p.Date != request.EffectiveLeavingDate.Format("2006-01-02") || p.Status != domain.StudentLeavingInProgress || p.ActorID != f.in.Actor.UserID || p.ActorRole != domain.RoleAdmin || !p.At.Equal(request.CreatedAt) {
			t.Fatalf("audit payload=%+v", p)
		}
	}
	if found != 1 {
		t.Fatalf("matching start audit count=%d, want 1", found)
	}
}

func TestStudentLeavingStartIntegrationValidation(t *testing.T) {
	cases := []struct {
		name   string
		change func(*testing.T, *leavingStartFixture, *service.StartStudentLeavingRequestInput)
		want   error
	}{
		{name: "admin sah tanpa rekening"},
		{name: "wali", change: func(_ *testing.T, _ *leavingStartFixture, in *service.StartStudentLeavingRequestInput) {
			in.Actor.Role = domain.RoleWaliKelas
		}, want: service.ErrForbidden},
		{name: "siswa", change: func(_ *testing.T, _ *leavingStartFixture, in *service.StartStudentLeavingRequestInput) {
			in.Actor.Role = domain.RoleSiswa
		}, want: service.ErrForbidden},
		{name: "student ID kosong", change: func(_ *testing.T, _ *leavingStartFixture, in *service.StartStudentLeavingRequestInput) {
			in.StudentID = uuid.Nil
		}, want: domain.ErrInvalidID},
		{name: "year ID kosong", change: func(_ *testing.T, _ *leavingStartFixture, in *service.StartStudentLeavingRequestInput) {
			in.AcademicYearID = uuid.Nil
		}, want: domain.ErrInvalidID},
		{name: "tanggal kosong", change: func(_ *testing.T, _ *leavingStartFixture, in *service.StartStudentLeavingRequestInput) {
			in.EffectiveLeavingDate = time.Time{}
		}, want: domain.ErrInvalidValue},
		{name: "student tidak ditemukan", change: func(_ *testing.T, _ *leavingStartFixture, in *service.StartStudentLeavingRequestInput) {
			in.StudentID = uuid.New()
		}, want: repository.ErrNotFound},
		{name: "year tidak ditemukan", change: func(_ *testing.T, _ *leavingStartFixture, in *service.StartStudentLeavingRequestInput) {
			in.AcademicYearID = uuid.New()
		}, want: repository.ErrNotFound},
		{name: "LEFT", change: func(t *testing.T, f *leavingStartFixture, _ *service.StartStudentLeavingRequestInput) {
			if _, err := f.db.Exec(`UPDATE students SET status='LEFT' WHERE id=$1`, f.in.StudentID); err != nil {
				t.Fatal(err)
			}
		}, want: domain.ErrInvalidTransition},
		{name: "GRADUATED", change: func(t *testing.T, f *leavingStartFixture, _ *service.StartStudentLeavingRequestInput) {
			if _, err := f.db.Exec(`UPDATE students SET status='GRADUATED' WHERE id=$1`, f.in.StudentID); err != nil {
				t.Fatal(err)
			}
		}, want: domain.ErrInvalidTransition},
		{name: "CLOSING", change: func(t *testing.T, f *leavingStartFixture, _ *service.StartStudentLeavingRequestInput) {
			y := f.year
			y.Status = domain.AcademicYearClosing
			if err := f.repos.AcademicYears.Update(context.Background(), y); err != nil {
				t.Fatal(err)
			}
		}, want: domain.ErrInvalidTransition},
		{name: "CLOSED", change: func(t *testing.T, f *leavingStartFixture, _ *service.StartStudentLeavingRequestInput) {
			y := f.year
			y.Status = domain.AcademicYearClosed
			if err := f.repos.AcademicYears.Update(context.Background(), y); err != nil {
				t.Fatal(err)
			}
		}, want: domain.ErrInvalidTransition},
		{name: "placement tidak ada", change: func(t *testing.T, f *leavingStartFixture, _ *service.StartStudentLeavingRequestInput) {
			if _, err := f.db.Exec(`DELETE FROM student_class_histories WHERE student_id=$1`, f.in.StudentID); err != nil {
				t.Fatal(err)
			}
		}, want: repository.ErrNotFound},
		{name: "kemarin WIB diterima", change: func(_ *testing.T, f *leavingStartFixture, in *service.StartStudentLeavingRequestInput) {
			in.EffectiveLeavingDate = in.EffectiveLeavingDate.AddDate(0, 0, -1)
			if in.EffectiveLeavingDate.Before(f.year.StartDate) {
				t.Skip("kemarin berada sebelum awal tahun ajaran pada hari pertama tahun")
			}
		}},
		{name: "tanggal awal tahun ajaran diterima", change: func(_ *testing.T, f *leavingStartFixture, in *service.StartStudentLeavingRequestInput) {
			in.EffectiveLeavingDate = f.year.StartDate
		}},
		{name: "besok WIB ditolak", change: func(_ *testing.T, _ *leavingStartFixture, in *service.StartStudentLeavingRequestInput) {
			in.EffectiveLeavingDate = in.EffectiveLeavingDate.AddDate(0, 0, 1)
		}, want: domain.ErrInvalidDateRange},
		{name: "sebelum tahun ajaran ditolak", change: func(_ *testing.T, f *leavingStartFixture, in *service.StartStudentLeavingRequestInput) {
			in.EffectiveLeavingDate = f.year.StartDate.AddDate(0, 0, -1)
		}, want: domain.ErrInvalidDateRange},
		{name: "batas akhir tahun ditolak", change: func(_ *testing.T, f *leavingStartFixture, in *service.StartStudentLeavingRequestInput) {
			in.EffectiveLeavingDate = f.year.EndDate
		}, want: domain.ErrInvalidDateRange},
		{name: "placement tidak berlaku pada tanggal lampau", change: func(t *testing.T, f *leavingStartFixture, in *service.StartStudentLeavingRequestInput) {
			in.EffectiveLeavingDate = f.year.StartDate
			if _, err := f.db.Exec(`UPDATE student_class_histories SET validity=daterange($2::date,$3::date,'[)') WHERE student_id=$1 AND academic_year_id=$4`, f.in.StudentID, f.year.StartDate.AddDate(0, 0, 1), f.year.EndDate, f.year.ID); err != nil {
				t.Fatal(err)
			}
		}, want: repository.ErrNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newLeavingStartFixture(t)
			in := f.in
			if tc.change != nil {
				tc.change(t, &f, &in)
			}
			before := f.preservedSnapshot(t)
			out, err := f.svc.Start(context.Background(), in)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Start error=%v, want %v", err, tc.want)
			}
			if after := f.preservedSnapshot(t); !reflect.DeepEqual(before, after) {
				t.Fatal("Start changed student/history/year/financial state")
			}
			if tc.want != nil {
				f.assertCounts(t, 0, 0)
				return
			}
			if out.Request == nil || out.Request.Status != domain.StudentLeavingInProgress {
				t.Fatalf("output=%+v", out)
			}
			f.assertCounts(t, 1, 1)
			f.assertAudit(t, *out.Request)
			got, err := f.svc.Get(context.Background(), service.GetStudentLeavingRequestInput{Actor: f.in.Actor, RequestID: out.Request.ID})
			if err != nil || got.Request == nil || got.Request.ID != out.Request.ID || got.Request.StudentID != f.in.StudentID {
				t.Fatalf("Get=%+v error=%v", got, err)
			}
		})
	}
}

func TestStudentLeavingStartIntegrationHistoricalProcesses(t *testing.T) {
	for _, status := range []domain.StudentLeavingRequestStatus{domain.StudentLeavingInProgress, domain.StudentLeavingCancelled, domain.StudentLeavingFinalized} {
		t.Run(string(status), func(t *testing.T) {
			f := newLeavingStartFixture(t)
			ctx := context.Background()
			old, err := domain.NewStudentLeavingRequest(f.in.StudentID, f.year.ID, f.in.EffectiveLeavingDate.AddDate(0, 0, -1), f.in.Actor.UserID)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.repos.StudentLeavingRequests.Create(ctx, old); err != nil {
				t.Fatal(err)
			}
			if status == domain.StudentLeavingCancelled {
				if err := old.Cancel(f.in.Actor.UserID, time.Now().UTC(), "Fixture dibatalkan"); err != nil {
					t.Fatal(err)
				}
			}
			if status == domain.StudentLeavingFinalized {
				if err := old.Finalize(f.in.Actor.UserID, time.Now().UTC()); err != nil {
					t.Fatal(err)
				}
			}
			if status != domain.StudentLeavingInProgress {
				if err := f.repos.StudentLeavingRequests.Update(ctx, old); err != nil {
					t.Fatal(err)
				}
			}
			before := f.preservedSnapshot(t)
			out, err := f.svc.Start(ctx, f.in)
			if status == domain.StudentLeavingInProgress {
				if !errors.Is(err, repository.ErrConflict) {
					t.Fatalf("duplicate error=%v", err)
				}
				f.assertCounts(t, 1, 0)
			} else {
				if err != nil {
					t.Fatal(err)
				}
				f.assertCounts(t, 2, 1)
				f.assertAudit(t, *out.Request)
			}
			persisted, err := f.repos.StudentLeavingRequests.GetByID(ctx, old.ID)
			if err != nil || persisted.Status != status {
				t.Fatalf("historical request changed: %+v %v", persisted, err)
			}
			if !reflect.DeepEqual(before, f.preservedSnapshot(t)) {
				t.Fatal("historical restart changed preserved data")
			}
		})
	}
}

func TestStudentLeavingStartIntegrationPreservesFinance(t *testing.T) {
	f := newLeavingStartFixture(t)
	ctx := context.Background()
	wali := newServiceScopeUser(t, domain.RoleWaliKelas)
	if err := f.repos.Users.Create(ctx, wali); err != nil {
		t.Fatal(err)
	}
	account, err := domain.NewSavingsAccount(f.in.StudentID, f.year.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.SavingsAccounts.Create(ctx, account); err != nil {
		t.Fatal(err)
	}
	transaction, err := domain.NewDeposit(account.ID, domain.Money(25000), f.in.EffectiveLeavingDate, wali.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.Transactions.Create(ctx, transaction); err != nil {
		t.Fatal(err)
	}
	if err := account.Settle(time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := f.repos.SavingsAccounts.Update(ctx, account); err != nil {
		t.Fatal(err)
	}
	settlement, err := domain.NewSavingsSettlement(account.ID, domain.SettlementStudentLeaving, domain.Money(25000), wali.ID, time.Now().UTC(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.Settlements.Create(ctx, settlement); err != nil {
		t.Fatal(err)
	}
	movement, err := domain.NewSettlementCashMovement(settlement.ID, domain.SettlementCashHandover, domain.Money(25000), wali.ID, time.Now().UTC(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.Settlements.CreateCashMovement(ctx, movement); err != nil {
		t.Fatal(err)
	}
	before := f.preservedSnapshot(t)
	out, err := f.svc.Start(ctx, f.in)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, f.preservedSnapshot(t)) {
		t.Fatal("Start changed historical financial or student data")
	}
	f.assertCounts(t, 1, 1)
	f.assertAudit(t, *out.Request)
}

type leavingStartRequestHook struct {
	repository.StudentLeavingRequestRepository
	afterCreate func(context.Context) error
}

func (r leavingStartRequestHook) Create(ctx context.Context, v domain.StudentLeavingRequest) error {
	if err := r.StudentLeavingRequestRepository.Create(ctx, v); err != nil {
		return err
	}
	if r.afterCreate != nil {
		return r.afterCreate(ctx)
	}
	return nil
}

type leavingStartAuditHook struct {
	repository.AuditRepository
	failure    error
	afterWrite bool
}

func (r leavingStartAuditHook) Create(ctx context.Context, v domain.AuditLog) error {
	if r.failure != nil && !r.afterWrite {
		return r.failure
	}
	if err := r.AuditRepository.Create(ctx, v); err != nil {
		return err
	}
	return r.failure
}

type leavingStartHookUOW struct {
	repository.UnitOfWork
	repos     repository.RepositorySet
	commitErr error
}

func (u leavingStartHookUOW) Repositories() repository.RepositorySet { return u.repos }
func (u leavingStartHookUOW) Commit() error {
	if u.commitErr != nil {
		return u.commitErr
	}
	return u.UnitOfWork.Commit()
}

type leavingStartHookManager struct {
	repository.UnitOfWorkManager
	afterCreate         func(context.Context) error
	auditErr, commitErr error
	auditAfterWrite     bool
}

func (m leavingStartHookManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	u, err := m.UnitOfWorkManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := u.Repositories()
	repos.StudentLeavingRequests = leavingStartRequestHook{StudentLeavingRequestRepository: repos.StudentLeavingRequests, afterCreate: m.afterCreate}
	repos.AuditLogs = leavingStartAuditHook{AuditRepository: repos.AuditLogs, failure: m.auditErr, afterWrite: m.auditAfterWrite}
	return leavingStartHookUOW{UnitOfWork: u, repos: repos, commitErr: m.commitErr}, nil
}
func (f leavingStartFixture) withHooks(t *testing.T, m leavingStartHookManager) service.StudentLeavingRequestService {
	t.Helper()
	m.UnitOfWorkManager = f.manager
	s, err := service.NewStudentLeavingRequestService(service.Dependencies{Repositories: f.repos, UOW: m})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestStudentLeavingStartIntegrationRollback(t *testing.T) {
	for _, stage := range []string{"sesudah INSERT", "audit sebelum tulis", "audit sesudah tulis", "commit"} {
		t.Run(stage, func(t *testing.T) {
			f := newLeavingStartFixture(t)
			failure := errors.New("kegagalan Start uji")
			m := leavingStartHookManager{}
			switch stage {
			case "sesudah INSERT":
				m.afterCreate = func(context.Context) error { return failure }
			case "audit sebelum tulis":
				m.auditErr = failure
			case "audit sesudah tulis":
				m.auditErr = failure
				m.auditAfterWrite = true
			case "commit":
				m.commitErr = failure
			}
			before := f.preservedSnapshot(t)
			svc := f.withHooks(t, m)
			if _, err := svc.Start(context.Background(), f.in); !errors.Is(err, failure) {
				t.Fatalf("error=%v", err)
			}
			f.assertCounts(t, 0, 0)
			if !reflect.DeepEqual(before, f.preservedSnapshot(t)) {
				t.Fatal("rollback left partial state")
			}
		})
	}
	t.Run("INSERT gagal FK", func(t *testing.T) {
		f := newLeavingStartFixture(t)
		in := f.in
		in.Actor.UserID = uuid.New()
		before := f.preservedSnapshot(t)
		_, err := f.svc.Start(context.Background(), in)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
			t.Fatalf("expected INSERT FK failure, got %v", err)
		}
		f.assertCounts(t, 0, 0)
		if !reflect.DeepEqual(before, f.preservedSnapshot(t)) {
			t.Fatal("failed INSERT changed state")
		}
	})
}

type leavingStartGate struct {
	entered, release chan struct{}
	once             sync.Once
}

func (g *leavingStartGate) hold(ctx context.Context) error {
	close(g.entered)
	select {
	case <-g.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (g *leavingStartGate) open() { g.once.Do(func() { close(g.release) }) }

func TestStudentLeavingStartIntegrationConcurrentStarts(t *testing.T) {
	for _, sameStudent := range []bool{true, false} {
		t.Run(map[bool]string{true: "siswa sama", false: "siswa berbeda tahun sama"}[sameStudent], func(t *testing.T) {
			f := newLeavingStartFixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			secondInput := f.in
			if !sameStudent {
				secondInput.StudentID = f.addStudent(t).ID
			}
			before := f.preservedSnapshot(t)
			gate := &leavingStartGate{entered: make(chan struct{}), release: make(chan struct{})}
			defer gate.open()
			svc := f.withHooks(t, leavingStartHookManager{afterCreate: gate.hold})
			type result struct {
				out service.StartStudentLeavingRequestOutput
				err error
			}
			first, second := make(chan result, 1), make(chan result, 1)
			go func() { out, err := svc.Start(ctx, f.in); first <- result{out, err} }()
			select {
			case <-gate.entered:
			case <-ctx.Done():
				t.Fatal("Start INSERT barrier not reached")
			}
			var pid int
			if err := f.db.QueryRowContext(ctx, `SELECT pid FROM pg_stat_activity WHERE datname=current_database() AND state='idle in transaction' AND query ILIKE '%INSERT INTO student_leaving_requests%'`).Scan(&pid); err != nil {
				t.Fatal(err)
			}
			go func() { out, err := f.svc.Start(ctx, secondInput); second <- result{out, err} }()
			waitForBlockedBackend(t, f.db, pid, "%FROM academic_years%FOR UPDATE%")
			gate.open()
			a, b := <-first, <-second
			if a.err != nil {
				t.Fatal(a.err)
			}
			f.assertAudit(t, *a.out.Request)
			if sameStudent {
				if !errors.Is(b.err, repository.ErrConflict) {
					t.Fatalf("second Start error=%v", b.err)
				}
				f.assertCounts(t, 1, 1)
			} else {
				if b.err != nil {
					t.Fatal(b.err)
				}
				if b.out.Request.StudentID != secondInput.StudentID || a.out.Request.StudentID != f.in.StudentID || b.out.Request.ID == a.out.Request.ID {
					t.Fatal("cross-student request mapping")
				}
				f.assertCounts(t, 2, 2)
				f.assertAudit(t, *b.out.Request)
			}
			if !reflect.DeepEqual(before, f.preservedSnapshot(t)) {
				t.Fatal("concurrent Starts changed preserved state")
			}
		})
	}
}

func TestStudentLeavingStartIntegrationRechecksAfterLockWait(t *testing.T) {
	for _, changeYear := range []bool{true, false} {
		t.Run(map[bool]string{true: "tahun berubah", false: "siswa berubah"}[changeYear], func(t *testing.T) {
			f := newLeavingStartFixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			tx, err := f.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			var pid int
			if err := tx.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
				t.Fatal(err)
			}
			if changeYear {
				if _, err := postgres.NewAcademicYearRepository(tx).GetByIDForUpdate(ctx, f.year.ID); err != nil {
					t.Fatal(err)
				}
				year := f.year
				year.Status = domain.AcademicYearClosing
				if err := postgres.NewAcademicYearRepository(tx).Update(ctx, year); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := postgres.NewStudentRepository(tx).GetByIDForUpdate(ctx, f.in.StudentID); err != nil {
					t.Fatal(err)
				}
				if _, err := tx.ExecContext(ctx, `UPDATE students SET status='LEFT' WHERE id=$1`, f.in.StudentID); err != nil {
					t.Fatal(err)
				}
			}
			result := make(chan error, 1)
			go func() { _, err := f.svc.Start(ctx, f.in); result <- err }()
			pattern := "%FROM students%FOR UPDATE%"
			if changeYear {
				pattern = "%FROM academic_years%FOR UPDATE%"
			}
			waitForBlockedBackend(t, f.db, pid, pattern)
			if changeYear {
				var id uuid.UUID
				if err := tx.QueryRowContext(ctx, `SELECT id FROM students WHERE id=$1 FOR UPDATE NOWAIT`, f.in.StudentID).Scan(&id); err != nil {
					t.Fatalf("Start locked student before year: %v", err)
				}
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			if err := <-result; !errors.Is(err, domain.ErrInvalidTransition) {
				t.Fatalf("Start after wait=%v", err)
			}
			f.assertCounts(t, 0, 0)
			student, err := f.repos.Students.GetByID(ctx, f.in.StudentID)
			if err != nil {
				t.Fatal(err)
			}
			year, err := f.repos.AcademicYears.GetByID(ctx, f.year.ID)
			if err != nil {
				t.Fatal(err)
			}
			if changeYear {
				if !student.IsActive() || year.Status != domain.AcademicYearClosing {
					t.Fatal("wrong year-first outcome")
				}
			} else if !student.IsLeft() || !year.IsOpen() {
				t.Fatal("wrong student-first outcome")
			}
		})
	}
}

package service_test

import (
	"context"
	"database/sql"
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
)

type transferDecisionFixture struct {
	transferRequestFixture
	request       domain.ClassTransferRequest
	effectiveDate time.Time
}

func newTransferDecisionFixture(t *testing.T) transferDecisionFixture {
	t.Helper()
	f := transferDecisionFixture{transferRequestFixture: newTransferRequestFixture(t), effectiveDate: time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)}
	out, err := f.svc.Request(context.Background(), f.input)
	if err != nil {
		t.Fatal(err)
	}
	f.request = *out.TransferRequest
	return f
}

func (f transferDecisionFixture) decide(ctx context.Context, svc service.ClassTransferService, approve bool) error {
	if approve {
		_, err := svc.Approve(ctx, service.ApproveTransferInput{Actor: f.admin, TransferRequestID: f.request.ID, EffectiveDate: f.effectiveDate})
		return err
	}
	_, err := svc.Reject(ctx, service.RejectTransferInput{Actor: f.admin, TransferRequestID: f.request.ID, RejectionReason: "Tidak sesuai penempatan"})
	return err
}

type transferDecisionSnapshot struct {
	request   domain.ClassTransferRequest
	histories []domain.StudentClassHistory
	student   domain.Student
	year      domain.AcademicYear
}

func (f transferDecisionFixture) snapshot(t *testing.T) transferDecisionSnapshot {
	t.Helper()
	ctx := context.Background()
	var s transferDecisionSnapshot
	var err error
	s.request, err = f.repos.ClassTransferRequests.GetByID(ctx, f.request.ID)
	if err != nil {
		t.Fatal(err)
	}
	s.histories, err = f.repos.StudentClassHistories.ListByStudentAndAcademicYear(ctx, f.input.StudentID, f.year.ID, repository.ListOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	s.student, err = f.repos.Students.GetByID(ctx, f.input.StudentID)
	if err != nil {
		t.Fatal(err)
	}
	s.year, err = f.repos.AcademicYears.GetByID(ctx, f.year.ID)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func (f transferDecisionFixture) assertDecision(t *testing.T, status domain.ClassTransferRequestStatus, yearStatus domain.AcademicYearStatus) {
	t.Helper()
	s := f.snapshot(t)
	if s.request.Status != status || s.year.Status != yearStatus {
		t.Fatalf("request/year status=%s/%s, want %s/%s", s.request.Status, s.year.Status, status, yearStatus)
	}
	if status == domain.TransferApproved {
		if len(s.histories) != 2 {
			t.Fatalf("histories=%+v", s.histories)
		}
		old, new := s.histories[0], s.histories[1]
		if old.ClassID != f.input.FromClassID || old.Period.To == nil || !old.Period.To.Equal(f.effectiveDate) || new.ClassID != f.input.ToClassID || !new.Period.From.Equal(f.effectiveDate) || new.Period.To == nil || !new.Period.To.Equal(f.year.EndDate) {
			t.Fatalf("partial/wrong placement: %+v", s.histories)
		}
	} else if len(s.histories) != 1 || s.histories[0].ClassID != f.input.FromClassID || s.histories[0].Period.To == nil || !s.histories[0].Period.To.Equal(f.year.EndDate) {
		t.Fatalf("placement changed: %+v", s.histories)
	}
}

func TestClassTransferDecisionIntegrationValidation(t *testing.T) {
	for _, approve := range []bool{true, false} {
		name := "reject"
		if approve {
			name = "approve"
		}
		t.Run(name, func(t *testing.T) {
			cases := []struct {
				name   string
				change func(*testing.T, *transferDecisionFixture)
				want   error
			}{
				{name: "sah"},
				{name: "role wali", change: func(_ *testing.T, f *transferDecisionFixture) { f.admin = f.input.Actor }, want: service.ErrForbidden},
				{name: "tidak ditemukan", change: func(_ *testing.T, f *transferDecisionFixture) { f.request.ID = uuid.New() }, want: repository.ErrNotFound},
				{name: "sudah terminal", change: func(t *testing.T, f *transferDecisionFixture) {
					if err := f.decide(context.Background(), f.svc, approve); err != nil {
						t.Fatal(err)
					}
				}, want: domain.ErrInvalidTransition},
				{name: "tahun closing", change: func(t *testing.T, f *transferDecisionFixture) {
					y := f.year
					y.Status = domain.AcademicYearClosing
					if err := f.repos.AcademicYears.Update(context.Background(), y); err != nil {
						t.Fatal(err)
					}
				}, want: domain.ErrInvalidTransition},
				{name: "tahun closed", change: func(t *testing.T, f *transferDecisionFixture) {
					y := f.year
					y.Status = domain.AcademicYearClosed
					if err := f.repos.AcademicYears.Update(context.Background(), y); err != nil {
						t.Fatal(err)
					}
				}, want: domain.ErrInvalidTransition},
			}
			if approve {
				cases = append(cases,
					struct {
						name   string
						change func(*testing.T, *transferDecisionFixture)
						want   error
					}{"sebelum tahun", func(_ *testing.T, f *transferDecisionFixture) { f.effectiveDate = f.year.StartDate.AddDate(0, 0, -1) }, domain.ErrInvalidDateRange},
					struct {
						name   string
						change func(*testing.T, *transferDecisionFixture)
						want   error
					}{"akhir tahun eksklusif", func(_ *testing.T, f *transferDecisionFixture) { f.effectiveDate = f.year.EndDate }, domain.ErrInvalidDateRange},
					struct {
						name   string
						change func(*testing.T, *transferDecisionFixture)
						want   error
					}{"placement asal berubah", func(t *testing.T, f *transferDecisionFixture) {
						if _, err := f.db.Exec(`UPDATE student_class_histories SET class_id=$1 WHERE student_id=$2`, f.input.ToClassID, f.input.StudentID); err != nil {
							t.Fatal(err)
						}
					}, service.ErrInvalidDependency},
					struct {
						name   string
						change func(*testing.T, *transferDecisionFixture)
						want   error
					}{"siswa LEFT", func(t *testing.T, f *transferDecisionFixture) {
						if _, err := f.db.Exec(`UPDATE students SET status='LEFT' WHERE id=$1`, f.input.StudentID); err != nil {
							t.Fatal(err)
						}
					}, domain.ErrInvalidTransition},
				)
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					f := newTransferDecisionFixture(t)
					originalID := f.request.ID
					if tc.change != nil {
						tc.change(t, &f)
					}
					inputID := f.request.ID
					f.request.ID = originalID
					before := f.snapshot(t)
					f.request.ID = inputID
					err := f.decide(context.Background(), f.svc, approve)
					if !errors.Is(err, tc.want) {
						t.Fatalf("decision error=%v, want %v", err, tc.want)
					}
					f.request.ID = originalID
					after := f.snapshot(t)
					if tc.want != nil {
						if !reflect.DeepEqual(before, after) {
							t.Fatalf("failed decision changed state: before=%+v after=%+v", before, after)
						}
						return
					}
					status := domain.TransferRejected
					if approve {
						status = domain.TransferApproved
					}
					f.assertDecision(t, status, domain.AcademicYearOpen)
					if !reflect.DeepEqual(before.student, after.student) {
						t.Fatal("decision changed student")
					}
					if !approve && !reflect.DeepEqual(before.histories, after.histories) {
						t.Fatal("Reject changed histories")
					}
				})
			}
		})
	}
}

// Hook hanya menahan transaksi atau menyuntikkan kegagalan pada repository PostgreSQL nyata.
type transferDecisionRequestHook struct {
	repository.ClassTransferRequestRepository
	afterLock, afterUpdate func(context.Context) error
}

func (r transferDecisionRequestHook) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.ClassTransferRequest, error) {
	v, err := r.ClassTransferRequestRepository.GetByIDForUpdate(ctx, id)
	if err == nil && r.afterLock != nil {
		err = r.afterLock(ctx)
	}
	return v, err
}
func (r transferDecisionRequestHook) Update(ctx context.Context, v domain.ClassTransferRequest) error {
	if err := r.ClassTransferRequestRepository.Update(ctx, v); err != nil {
		return err
	}
	if r.afterUpdate != nil {
		return r.afterUpdate(ctx)
	}
	return nil
}

type transferDecisionYearHook struct {
	repository.AcademicYearRepository
	afterLock func(context.Context) error
}

func (r transferDecisionYearHook) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.AcademicYear, error) {
	v, err := r.AcademicYearRepository.GetByIDForUpdate(ctx, id)
	if err == nil && r.afterLock != nil {
		err = r.afterLock(ctx)
	}
	return v, err
}

type transferDecisionHistoryHook struct {
	repository.StudentClassHistoryRepository
	stage   string
	failure error
}

func (r transferDecisionHistoryHook) CloseCurrentByStudentAndAcademicYear(ctx context.Context, sid, yid uuid.UUID, at time.Time) (domain.StudentClassHistory, error) {
	v, err := r.StudentClassHistoryRepository.CloseCurrentByStudentAndAcademicYear(ctx, sid, yid, at)
	if err == nil && r.stage == "close" {
		err = r.failure
	}
	return v, err
}
func (r transferDecisionHistoryHook) Create(ctx context.Context, h domain.StudentClassHistory) error {
	if err := r.StudentClassHistoryRepository.Create(ctx, h); err != nil {
		return err
	}
	if r.stage == "create" {
		return r.failure
	}
	return nil
}

type transferDecisionHookManager struct {
	repository.UnitOfWorkManager
	afterRequestLock, afterRequestUpdate, afterYearLock func(context.Context) error
	historyStage                                        string
	failure                                             error
}

func (m transferDecisionHookManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	u, err := m.UnitOfWorkManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := u.Repositories()
	repos.ClassTransferRequests = transferDecisionRequestHook{ClassTransferRequestRepository: repos.ClassTransferRequests, afterLock: m.afterRequestLock, afterUpdate: m.afterRequestUpdate}
	repos.AcademicYears = transferDecisionYearHook{AcademicYearRepository: repos.AcademicYears, afterLock: m.afterYearLock}
	repos.StudentClassHistories = transferDecisionHistoryHook{StudentClassHistoryRepository: repos.StudentClassHistories, stage: m.historyStage, failure: m.failure}
	return transferRequestHookUOW{UnitOfWork: u, repos: repos}, nil
}
func (f transferDecisionFixture) serviceWithHooks(t *testing.T, m transferDecisionHookManager) service.ClassTransferService {
	t.Helper()
	m.UnitOfWorkManager = f.manager
	svc, err := service.NewClassTransferService(service.Dependencies{Repositories: f.repos, UOW: m})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestClassTransferDecisionIntegrationRollback(t *testing.T) {
	for _, stage := range []string{"close", "create", "request update"} {
		t.Run(stage, func(t *testing.T) {
			f := newTransferDecisionFixture(t)
			before := f.snapshot(t)
			failure := errors.New("kegagalan keputusan uji")
			hooks := transferDecisionHookManager{historyStage: stage, failure: failure}
			if stage == "request update" {
				hooks.afterRequestUpdate = func(context.Context) error { return failure }
			}
			svc := f.serviceWithHooks(t, hooks)
			if err := f.decide(context.Background(), svc, true); !errors.Is(err, failure) {
				t.Fatalf("error=%v", err)
			}
			if after := f.snapshot(t); !reflect.DeepEqual(before, after) {
				t.Fatalf("partial state after rollback: %+v", after)
			}
		})
	}
}

type transferDecisionGate struct {
	entered, release chan struct{}
	once             sync.Once
}

func newTransferDecisionGate() *transferDecisionGate {
	return &transferDecisionGate{entered: make(chan struct{}), release: make(chan struct{})}
}
func (g *transferDecisionGate) hold(ctx context.Context) error {
	close(g.entered)
	select {
	case <-g.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (g *transferDecisionGate) open() { g.once.Do(func() { close(g.release) }) }
func (g *transferDecisionGate) wait(t *testing.T, ctx context.Context) {
	t.Helper()
	select {
	case <-g.entered:
	case <-ctx.Done():
		t.Fatal("barrier not reached")
	}
}
func decisionIdleBackend(t *testing.T, ctx context.Context, db *sql.DB, pattern string) int {
	t.Helper()
	var pid int
	if err := db.QueryRowContext(ctx, `SELECT pid FROM pg_stat_activity WHERE datname=current_database() AND state='idle in transaction' AND query ILIKE $1`, pattern).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	return pid
}

func TestClassTransferDecisionIntegrationStartClosingBothOrders(t *testing.T) {
	for _, approve := range []bool{true, false} {
		for _, yearFirst := range []bool{true, false} {
			name := "reject"
			if approve {
				name = "approve"
			}
			if yearFirst {
				name += " tahun dahulu"
			} else {
				name += " keputusan dahulu"
			}
			t.Run(name, func(t *testing.T) {
				f := newTransferDecisionFixture(t)
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				gate := newTransferDecisionGate()
				defer gate.open()
				hooks := transferDecisionHookManager{UnitOfWorkManager: f.manager}
				if yearFirst {
					hooks.afterYearLock = gate.hold
				} else {
					hooks.afterRequestLock = gate.hold
				}
				decisionService := f.serviceWithHooks(t, hooks)
				yearService, err := service.NewAcademicYearService(service.Dependencies{Repositories: f.repos, UOW: hooks})
				if err != nil {
					t.Fatal(err)
				}
				decision, closing := make(chan error, 1), make(chan error, 1)
				decide := func() { decision <- f.decide(ctx, decisionService, approve) }
				startClosing := func() {
					_, err := yearService.StartClosing(ctx, service.StartClosingAcademicYearInput{Actor: f.admin, AcademicYearID: f.year.ID})
					closing <- err
				}
				if yearFirst {
					go startClosing()
					gate.wait(t, ctx)
					pid := decisionIdleBackend(t, ctx, f.db, "%FROM academic_years%FOR UPDATE%")
					go decide()
					waitForBlockedBackend(t, f.db, pid, "%FROM academic_years%FOR SHARE%")
					gate.open()
					if err := <-closing; !errors.Is(err, service.ErrAcademicYearHasPendingTransfers) {
						t.Fatalf("StartClosing=%v", err)
					}
				} else {
					go decide()
					gate.wait(t, ctx)
					pid := decisionIdleBackend(t, ctx, f.db, "%FROM class_transfer_requests%FOR UPDATE%")
					go startClosing()
					waitForBlockedBackend(t, f.db, pid, "%FROM academic_years%FOR UPDATE%")
					gate.open()
					if err := <-closing; err != nil {
						t.Fatalf("StartClosing=%v", err)
					}
				}
				if err := <-decision; err != nil {
					t.Fatalf("decision=%v", err)
				}
				status := domain.TransferRejected
				if approve {
					status = domain.TransferApproved
				}
				yearStatus := domain.AcademicYearClosing
				if yearFirst {
					yearStatus = domain.AcademicYearOpen
				}
				f.assertDecision(t, status, yearStatus)
			})
		}
	}
}

func TestClassTransferDecisionIntegrationCompetingDecisions(t *testing.T) {
	for _, pair := range []struct {
		name          string
		first, second bool
	}{{"dua approve", true, true}, {"approve lalu reject", true, false}, {"reject lalu approve", false, true}} {
		t.Run(pair.name, func(t *testing.T) {
			f := newTransferDecisionFixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			gate := newTransferDecisionGate()
			defer gate.open()
			svc := f.serviceWithHooks(t, transferDecisionHookManager{afterRequestLock: gate.hold})
			first, second := make(chan error, 1), make(chan error, 1)
			go func() { first <- f.decide(ctx, svc, pair.first) }()
			gate.wait(t, ctx)
			pid := decisionIdleBackend(t, ctx, f.db, "%FROM class_transfer_requests%FOR UPDATE%")
			go func() { second <- f.decide(ctx, f.svc, pair.second) }()
			pattern := "%FROM class_transfer_requests%FOR UPDATE%"
			if pair.first && pair.second {
				pattern = "%FROM students%FOR UPDATE%"
			}
			waitForBlockedBackend(t, f.db, pid, pattern)
			gate.open()
			if err := <-first; err != nil {
				t.Fatalf("first=%v", err)
			}
			if err := <-second; !errors.Is(err, domain.ErrInvalidTransition) {
				t.Fatalf("second=%v", err)
			}
			status := domain.TransferRejected
			if pair.first {
				status = domain.TransferApproved
			}
			f.assertDecision(t, status, domain.AcademicYearOpen)
		})
	}
}

func TestClassTransferDecisionIntegrationStudentStatusRechecked(t *testing.T) {
	f := newTransferDecisionFixture(t)
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
	if _, err := postgres.NewStudentRepository(tx).GetByIDForUpdate(ctx, f.input.StudentID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE students SET status='LEFT' WHERE id=$1`, f.input.StudentID); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { result <- f.decide(ctx, f.svc, true) }()
	waitForBlockedBackend(t, f.db, pid, "%FROM students%FOR UPDATE%")
	var id uuid.UUID
	if err := tx.QueryRowContext(ctx, `SELECT id FROM class_transfer_requests WHERE id=$1 FOR UPDATE NOWAIT`, f.request.ID).Scan(&id); err != nil {
		t.Fatalf("request locked before student: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-result; !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("Approve=%v", err)
	}
	f.assertDecision(t, domain.TransferPending, domain.AcademicYearOpen)
	if s := f.snapshot(t); s.student.Status != domain.StudentLeft {
		t.Fatalf("student=%s", s.student.Status)
	}
}

func TestClassTransferDecisionIntegrationRejectDoesNotLockStudent(t *testing.T) {
	f := newTransferDecisionFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := postgres.NewStudentRepository(tx).GetByIDForUpdate(ctx, f.input.StudentID); err != nil {
		t.Fatal(err)
	}
	if err := f.decide(ctx, f.svc, false); err != nil {
		t.Fatalf("Reject waited for irrelevant student lock: %v", err)
	}
	f.assertDecision(t, domain.TransferRejected, domain.AcademicYearOpen)
}

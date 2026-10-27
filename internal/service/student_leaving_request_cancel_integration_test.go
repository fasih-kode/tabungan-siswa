package service_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func TestStudentLeavingCancelIntegration(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T, leavingStartFixture)
	}{
		{name: "sebelum rekening dibuat", run: testCancelBeforeAccount},
		{name: "rekening masih terbuka", run: testCancelOpenAccount},
		{name: "settlement nol", run: testCancelZeroSettlement},
		{name: "settlement positif dan penyerahan lengkap", run: testCancelPositiveHandover},
		{name: "penyerahan belum dilakukan ditolak", run: testCancelNoHandover},
		{name: "penyerahan sebagian ditolak", run: testCancelPartialHandover},
		{name: "penyerahan berlebih ditolak", run: testCancelExcessHandover},
		{name: "relasi keuangan tidak valid ditolak", run: testCancelInvalidFinancialLink},
		{name: "proses FINALIZED ditolak", run: testCancelFinalized},
		{name: "pembatalan ulang ditolak", run: testCancelRepeated},
		{name: "Wali dan Siswa tidak berwenang", run: testCancelUnauthorized},
		{name: "alasan tidak sah ditolak", run: testCancelInvalidReason},
		{name: "ID proses lain tidak mengubah data", run: testCancelMissingRequest},
		{name: "proses siswa lain tetap utuh", run: testCancelOtherRequestIsolation},
		{name: "settlement historis NULL dipertahankan", run: testCancelHistoricalNullSettlement},
		{name: "rekening lintas tahun tetap diperiksa dan dipertahankan", run: testCancelAccountsAcrossYears},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { tc.run(t, newLeavingCancelFixture(t)) })
	}
}

func TestStudentLeavingCancelIntegrationRollback(t *testing.T) {
	for _, stage := range []string{"settlement read", "cash movement read", "request update", "audit after insert", "commit"} {
		t.Run(stage, func(t *testing.T) {
			f := newLeavingCancelFixture(t)
			var account *domain.SavingsAccount
			if stage == "cash movement read" {
				created := createCancelAccount(t, f, f.year.ID)
				account = &created
			}
			request := startLeavingRequestForCancel(t, f)
			if stage == "cash movement read" {
				createLinkedCancelSettlement(t, f, request, *account, 0)
			}
			before := cancelSnapshot(t, f.db)
			injected := errors.New("injected cancellation failure")
			manager := cancelFailureManager{UnitOfWorkManager: f.manager, stage: stage, failure: injected}
			svc, err := service.NewStudentLeavingRequestService(service.Dependencies{Repositories: f.repos, UOW: manager})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := svc.Cancel(context.Background(), cancelInput(f, request.ID, "uji rollback")); !errors.Is(err, injected) {
				t.Fatalf("Cancel() error=%v, want injected failure", err)
			}
			if after := cancelSnapshot(t, f.db); !reflect.DeepEqual(before, after) {
				t.Fatalf("failure at %s left partial database changes", stage)
			}
		})
	}
}

func TestStudentLeavingCancelConcurrency(t *testing.T) {
	t.Run("Cancel dan penyerahan yang lebih dahulu", func(t *testing.T) {
		f, request, settlement, waliActor := newCancelPartialHandoverFixture(t)
		gate := newCancelConcurrencyGate()
		manager := cancelConcurrencyManager{UnitOfWorkManager: f.manager, pause: "account", gate: gate}
		settlementSvc, err := service.NewSettlementService(service.Dependencies{Repositories: f.repos, UOW: manager})
		if err != nil {
			t.Fatal(err)
		}
		confirmResult := make(chan error, 1)
		go func() {
			_, confirmErr := settlementSvc.ConfirmHandover(context.Background(), service.ConfirmSettlementHandoverInput{Actor: waliActor, SettlementID: settlement.ID, Confirm: true})
			confirmResult <- confirmErr
		}()
		waitCancelGate(t, gate)
		blockerPID := cancelIdleTransactionPID(t, f.db)
		cancelResult := make(chan error, 1)
		go func() {
			_, cancelErr := f.svc.Cancel(context.Background(), cancelInput(f, request.ID, "uji urutan penyerahan"))
			cancelResult <- cancelErr
		}()
		waitForCancelBlocker(t, f.db, blockerPID)
		gate.release()
		if err := <-confirmResult; err != nil {
			t.Fatalf("ConfirmHandover(): %v", err)
		}
		if err := <-cancelResult; err != nil {
			t.Fatalf("Cancel() setelah penyerahan commit: %v", err)
		}
	})

	t.Run("Cancel lebih dahulu menolak saat penyerahan belum lengkap", func(t *testing.T) {
		f, request, settlement, waliActor := newCancelPartialHandoverFixture(t)
		gate := newCancelConcurrencyGate()
		cancelSvc, err := service.NewStudentLeavingRequestService(service.Dependencies{Repositories: f.repos, UOW: cancelConcurrencyManager{UnitOfWorkManager: f.manager, pause: "finance", gate: gate}})
		if err != nil {
			t.Fatal(err)
		}
		cancelResult := make(chan error, 1)
		go func() {
			_, cancelErr := cancelSvc.Cancel(context.Background(), cancelInput(f, request.ID, "uji penyerahan tertunda"))
			cancelResult <- cancelErr
		}()
		waitCancelGate(t, gate)
		blockerPID := cancelIdleTransactionPID(t, f.db)
		settlementSvc, err := service.NewSettlementService(service.Dependencies{Repositories: f.repos, UOW: f.manager})
		if err != nil {
			t.Fatal(err)
		}
		confirmResult := make(chan error, 1)
		go func() {
			_, confirmErr := settlementSvc.ConfirmHandover(context.Background(), service.ConfirmSettlementHandoverInput{Actor: waliActor, SettlementID: settlement.ID, Confirm: true})
			confirmResult <- confirmErr
		}()
		waitForCancelBlocker(t, f.db, blockerPID)
		gate.release()
		if err := <-cancelResult; !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("Cancel() error=%v, want incomplete-handover blocker", err)
		}
		if err := <-confirmResult; err != nil {
			t.Fatalf("ConfirmHandover() after rejected Cancel: %v", err)
		}
		got, err := f.repos.StudentLeavingRequests.GetByID(context.Background(), request.ID)
		if err != nil || got.Status != domain.StudentLeavingInProgress {
			t.Fatalf("request status=%s err=%v", got.Status, err)
		}
	})

	t.Run("dua Cancel hanya mengubah satu kali", func(t *testing.T) {
		f := newLeavingCancelFixture(t)
		createCancelAccount(t, f, f.year.ID)
		request := startLeavingRequestForCancel(t, f)
		gate := newCancelConcurrencyGate()
		cancelSvc, err := service.NewStudentLeavingRequestService(service.Dependencies{Repositories: f.repos, UOW: cancelConcurrencyManager{UnitOfWorkManager: f.manager, pause: "finance", gate: gate}})
		if err != nil {
			t.Fatal(err)
		}
		first := make(chan error, 1)
		go func() {
			_, e := cancelSvc.Cancel(context.Background(), cancelInput(f, request.ID, "cancel pertama"))
			first <- e
		}()
		waitCancelGate(t, gate)
		blockerPID := cancelIdleTransactionPID(t, f.db)
		second := make(chan error, 1)
		go func() {
			_, e := f.svc.Cancel(context.Background(), cancelInput(f, request.ID, "cancel kedua"))
			second <- e
		}()
		waitForCancelBlocker(t, f.db, blockerPID)
		gate.release()
		if err := <-first; err != nil {
			t.Fatalf("Cancel pertama: %v", err)
		}
		if err := <-second; !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("Cancel kedua error=%v", err)
		}
		assertActionCount(t, f.db, "STUDENT_LEAVING_CANCELLED", 1)
	})

	t.Run("Start baru menunggu Cancel lalu dapat dimulai", func(t *testing.T) {
		f := newLeavingCancelFixture(t)
		request := startLeavingRequestForCancel(t, f)
		gate := newCancelConcurrencyGate()
		cancelSvc, err := service.NewStudentLeavingRequestService(service.Dependencies{Repositories: f.repos, UOW: cancelConcurrencyManager{UnitOfWorkManager: f.manager, pause: "finance", gate: gate}})
		if err != nil {
			t.Fatal(err)
		}
		cancelResult := make(chan error, 1)
		go func() {
			_, e := cancelSvc.Cancel(context.Background(), cancelInput(f, request.ID, "uji start"))
			cancelResult <- e
		}()
		waitCancelGate(t, gate)
		blockerPID := cancelIdleTransactionPID(t, f.db)
		startResult := make(chan error, 1)
		go func() { _, e := f.svc.Start(context.Background(), f.in); startResult <- e }()
		waitForCancelBlocker(t, f.db, blockerPID)
		gate.release()
		if err := <-cancelResult; err != nil {
			t.Fatalf("Cancel(): %v", err)
		}
		if err := <-startResult; err != nil {
			t.Fatalf("Start() after Cancel: %v", err)
		}
	})

	t.Run("INSERT rekening ditolak sebelum Cancel dan menunggu sesudahnya", func(t *testing.T) {
		t.Run("sebelum", func(t *testing.T) {
			f := newLeavingCancelFixture(t)
			request := startLeavingRequestForCancel(t, f)
			closeCancelFixtureYear(t, f)
			newYear := newServiceScopeAcademicYearForStartYear(t, "Tahun INSERT sebelum cancel "+uuid.NewString(), f.year.StartDate.Year()+1)
			if err := f.repos.AcademicYears.Create(context.Background(), newYear); err != nil {
				t.Fatal(err)
			}
			account, err := domain.NewSavingsAccount(f.in.StudentID, newYear.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.repos.SavingsAccounts.Create(context.Background(), account); err == nil {
				t.Fatal("INSERT rekening selama proses aktif berhasil")
			}
			got, err := f.repos.StudentLeavingRequests.GetByID(context.Background(), request.ID)
			if err != nil || got.Status != domain.StudentLeavingInProgress {
				t.Fatalf("request status=%s err=%v", got.Status, err)
			}
		})
		t.Run("sesudah", func(t *testing.T) {
			f := newLeavingCancelFixture(t)
			request := startLeavingRequestForCancel(t, f)
			closeCancelFixtureYear(t, f)
			newYear := newServiceScopeAcademicYearForStartYear(t, "Tahun INSERT sesudah cancel "+uuid.NewString(), f.year.StartDate.Year()+1)
			if err := f.repos.AcademicYears.Create(context.Background(), newYear); err != nil {
				t.Fatal(err)
			}
			gate := newCancelConcurrencyGate()
			cancelSvc, err := service.NewStudentLeavingRequestService(service.Dependencies{Repositories: f.repos, UOW: cancelConcurrencyManager{UnitOfWorkManager: f.manager, pause: "finance", gate: gate}})
			if err != nil {
				t.Fatal(err)
			}
			cancelResult := make(chan error, 1)
			go func() {
				_, e := cancelSvc.Cancel(context.Background(), cancelInput(f, request.ID, "uji insert"))
				cancelResult <- e
			}()
			waitCancelGate(t, gate)
			blockerPID := cancelIdleTransactionPID(t, f.db)
			account, err := domain.NewSavingsAccount(f.in.StudentID, newYear.ID)
			if err != nil {
				t.Fatal(err)
			}
			insertResult := make(chan error, 1)
			go func() { insertResult <- f.repos.SavingsAccounts.Create(context.Background(), account) }()
			waitForCancelBlocker(t, f.db, blockerPID)
			gate.release()
			if err := <-cancelResult; err != nil {
				t.Fatalf("Cancel(): %v", err)
			}
			if err := <-insertResult; err != nil {
				t.Fatalf("INSERT after Cancel: %v", err)
			}
		})
	})
}

func closeCancelFixtureYear(t *testing.T, f leavingStartFixture) {
	t.Helper()
	ctx := context.Background()
	year, err := f.repos.AcademicYears.GetByID(ctx, f.year.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := year.StartClosing(); err != nil {
		t.Fatal(err)
	}
	if err := f.repos.AcademicYears.Update(ctx, year); err != nil {
		t.Fatal(err)
	}
	year, err = f.repos.AcademicYears.GetByID(ctx, f.year.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := year.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.repos.AcademicYears.Update(ctx, year); err != nil {
		t.Fatal(err)
	}
}

func newCancelPartialHandoverFixture(t *testing.T) (leavingStartFixture, domain.StudentLeavingRequest, domain.SavingsSettlement, service.Actor) {
	t.Helper()
	f := newLeavingCancelFixture(t)
	account := createCancelAccount(t, f, f.year.ID)
	wali, waliActor := addCancelWali(t, f)
	mustCancelDeposit(t, f, wali, account, 50_000)
	request := startLeavingRequestForCancel(t, f)
	settlement := createLinkedCancelSettlement(t, f, request, account, 50_000)
	insertCancelMovement(t, f, settlement, domain.SettlementCashHandover, 20_000)
	return f, request, settlement, waliActor
}

type cancelConcurrencyGate struct {
	reached   chan struct{}
	releaseCh chan struct{}
	once      sync.Once
}

func newCancelConcurrencyGate() *cancelConcurrencyGate {
	return &cancelConcurrencyGate{reached: make(chan struct{}), releaseCh: make(chan struct{})}
}
func (g *cancelConcurrencyGate) pause() { g.once.Do(func() { close(g.reached); <-g.releaseCh }) }
func (g *cancelConcurrencyGate) release() {
	select {
	case <-g.releaseCh:
	default:
		close(g.releaseCh)
	}
}

type cancelConcurrencyManager struct {
	repository.UnitOfWorkManager
	pause string
	gate  *cancelConcurrencyGate
}

func (m cancelConcurrencyManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	uow, err := m.UnitOfWorkManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := uow.Repositories()
	if m.pause == "finance" {
		repos.StudentLeavingRequests = cancelPauseIntegrityRepository{StudentLeavingRequestRepository: repos.StudentLeavingRequests, gate: m.gate}
	}
	if m.pause == "account" {
		repos.SavingsAccounts = cancelPauseAccountRepository{SavingsAccountRepository: repos.SavingsAccounts, gate: m.gate}
	}
	return cancelPauseUOW{UnitOfWork: uow, repos: repos}, nil
}

type cancelPauseUOW struct {
	repository.UnitOfWork
	repos repository.RepositorySet
}

func (u cancelPauseUOW) Repositories() repository.RepositorySet { return u.repos }

type cancelPauseIntegrityRepository struct {
	repository.StudentLeavingRequestRepository
	gate *cancelConcurrencyGate
}

func (r cancelPauseIntegrityRepository) ListFinancialLinkIssues(ctx context.Context, id uuid.UUID) ([]repository.StudentLeavingFinancialLinkIssue, error) {
	issues, err := r.StudentLeavingRequestRepository.ListFinancialLinkIssues(ctx, id)
	if err == nil {
		r.gate.pause()
	}
	return issues, err
}

type cancelPauseAccountRepository struct {
	repository.SavingsAccountRepository
	gate *cancelConcurrencyGate
}

func (r cancelPauseAccountRepository) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.SavingsAccount, error) {
	account, err := r.SavingsAccountRepository.GetByIDForUpdate(ctx, id)
	if err == nil {
		r.gate.pause()
	}
	return account, err
}

func waitCancelGate(t *testing.T, gate *cancelConcurrencyGate) {
	t.Helper()
	select {
	case <-gate.reached:
	case <-time.After(8 * time.Second):
		t.Fatal("timeout menunggu barrier transaksi")
	}
}

func cancelIdleTransactionPID(t *testing.T, db *sql.DB) int {
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
	t.Fatal("tidak menemukan backend Cancel yang idle dalam transaksi")
	return 0
}

func waitForCancelBlocker(t *testing.T, db *sql.DB, blockerPID int) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		var blocked bool
		err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND pid<>pg_backend_pid() AND $1=ANY(pg_blocking_pids(pid)))`, blockerPID).Scan(&blocked)
		if err != nil {
			t.Fatalf("amati pg_blocking_pids: %v", err)
		}
		if blocked {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("tidak melihat transaksi diblokir oleh backend %d", blockerPID)
}

func assertActionCount(t *testing.T, db *sql.DB, action string, want int) {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM audit_logs WHERE action=$1`, action).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("audit action %s count=%d, want %d", action, count, want)
	}
}

type cancelFailureManager struct {
	repository.UnitOfWorkManager
	stage   string
	failure error
}

func (m cancelFailureManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	uow, err := m.UnitOfWorkManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := uow.Repositories()
	switch m.stage {
	case "settlement read":
		repos.Settlements = cancelSettlementFailureRepository{SettlementRepository: repos.Settlements, listErr: m.failure}
	case "cash movement read":
		repos.Settlements = cancelSettlementFailureRepository{SettlementRepository: repos.Settlements, movementErr: m.failure}
	case "request update":
		repos.StudentLeavingRequests = cancelRequestFailureRepository{StudentLeavingRequestRepository: repos.StudentLeavingRequests, updateErr: m.failure}
	case "audit after insert":
		repos.AuditLogs = cancelAuditFailureRepository{AuditRepository: repos.AuditLogs, afterInsertErr: m.failure}
	}
	return cancelFailureUOW{UnitOfWork: uow, repos: repos, commitErr: func() error {
		if m.stage == "commit" {
			return m.failure
		}
		return nil
	}()}, nil
}

type cancelFailureUOW struct {
	repository.UnitOfWork
	repos     repository.RepositorySet
	commitErr error
}

func (u cancelFailureUOW) Repositories() repository.RepositorySet { return u.repos }
func (u cancelFailureUOW) Commit() error {
	if u.commitErr != nil {
		return u.commitErr
	}
	return u.UnitOfWork.Commit()
}

type cancelSettlementFailureRepository struct {
	repository.SettlementRepository
	listErr, movementErr error
}

func (r cancelSettlementFailureRepository) ListByStudentLeavingRequest(ctx context.Context, id uuid.UUID) ([]domain.SavingsSettlement, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.SettlementRepository.ListByStudentLeavingRequest(ctx, id)
}
func (r cancelSettlementFailureRepository) ListCashMovements(ctx context.Context, id uuid.UUID) ([]domain.SettlementCashMovement, error) {
	if r.movementErr != nil {
		return nil, r.movementErr
	}
	return r.SettlementRepository.ListCashMovements(ctx, id)
}

type cancelRequestFailureRepository struct {
	repository.StudentLeavingRequestRepository
	updateErr error
}

func (r cancelRequestFailureRepository) Update(ctx context.Context, request domain.StudentLeavingRequest) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	return r.StudentLeavingRequestRepository.Update(ctx, request)
}

type cancelAuditFailureRepository struct {
	repository.AuditRepository
	afterInsertErr error
}

func (r cancelAuditFailureRepository) Create(ctx context.Context, audit domain.AuditLog) error {
	if err := r.AuditRepository.Create(ctx, audit); err != nil {
		return err
	}
	return r.afterInsertErr
}

func testCancelBeforeAccount(t *testing.T, f leavingStartFixture) {
	request := startLeavingRequestForCancel(t, f)
	before := cancelSnapshot(t, f.db)
	out, err := f.svc.Cancel(context.Background(), cancelInput(f, request.ID, "tidak jadi pindah"))
	if err != nil || out.Request == nil || out.Request.Status != domain.StudentLeavingCancelled {
		t.Fatalf("Cancel()=%+v, %v", out, err)
	}
	assertCancelledAudit(t, f.db, *out.Request, f.in.Actor.UserID)
	assertCancellationPreservedNonLifecycleData(t, before, cancelSnapshot(t, f.db))
	student, err := f.repos.Students.GetByID(context.Background(), f.in.StudentID)
	if err != nil || !student.IsActive() || student.LeavingDate == nil || !student.LeavingDate.Equal(f.year.StartDate.AddDate(-1, 0, 0)) {
		t.Fatalf("student status/leaving date changed: %+v, %v", student, err)
	}
	if _, err := f.repos.StudentClassHistories.GetCurrentByStudentAndAcademicYear(context.Background(), student.ID, f.year.ID, f.in.EffectiveLeavingDate); err != nil {
		t.Fatalf("placement no longer current: %v", err)
	}
}

func testCancelOpenAccount(t *testing.T, f leavingStartFixture) {
	account := createCancelAccount(t, f, f.year.ID)
	request := startLeavingRequestForCancel(t, f)
	if _, err := f.svc.Cancel(context.Background(), cancelInput(f, request.ID, "pindah dibatalkan")); err != nil {
		t.Fatal(err)
	}
	got, err := f.repos.SavingsAccounts.GetByID(context.Background(), account.ID)
	if err != nil || !got.IsOpen() {
		t.Fatalf("open account changed: %+v, %v", got, err)
	}
}

func testCancelZeroSettlement(t *testing.T, f leavingStartFixture) {
	account := createCancelAccount(t, f, f.year.ID)
	request := startLeavingRequestForCancel(t, f)
	settlement := createLinkedCancelSettlement(t, f, request, account, 0)
	before := cancelSnapshot(t, f.db)
	out, err := f.svc.Cancel(context.Background(), cancelInput(f, request.ID, "saldo nol, proses batal"))
	if err != nil || out.Request == nil || out.Request.Status != domain.StudentLeavingCancelled {
		t.Fatalf("Cancel()=%+v, %v", out, err)
	}
	got, err := f.repos.Settlements.GetByID(context.Background(), settlement.ID)
	if err != nil || got.Status != domain.SettlementCompleted || got.Amount != 0 {
		t.Fatalf("zero settlement changed: %+v, %v", got, err)
	}
	assertCancellationPreservedNonLifecycleData(t, before, cancelSnapshot(t, f.db))
}

func testCancelPositiveHandover(t *testing.T, f leavingStartFixture) {
	account := createCancelAccount(t, f, f.year.ID)
	wali, waliActor := addCancelWali(t, f)
	deposit := mustCancelDeposit(t, f, wali, account, 50_000)
	request := startLeavingRequestForCancel(t, f)
	settlement := createLinkedCancelSettlement(t, f, request, account, 50_000)
	settlementService, err := service.NewSettlementService(service.Dependencies{Repositories: f.repos, UOW: f.manager})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := settlementService.ConfirmHandover(context.Background(), service.ConfirmSettlementHandoverInput{Actor: waliActor, SettlementID: settlement.ID, Confirm: true}); err != nil {
		t.Fatalf("ConfirmHandover(): %v", err)
	}
	before := cancelSnapshot(t, f.db)
	out, err := f.svc.Cancel(context.Background(), cancelInput(f, request.ID, "penyerahan selesai, batal pindah"))
	if err != nil || out.Request == nil || out.Request.Status != domain.StudentLeavingCancelled {
		t.Fatalf("Cancel()=%+v, %v", out, err)
	}
	assertCancelledAudit(t, f.db, *out.Request, f.in.Actor.UserID)
	assertCancellationPreservedNonLifecycleData(t, before, cancelSnapshot(t, f.db))
	if _, err := f.repos.Transactions.GetByID(context.Background(), deposit.ID); err != nil {
		t.Fatalf("deposit disappeared: %v", err)
	}
}

func testCancelNoHandover(t *testing.T, f leavingStartFixture) {
	account := createCancelAccount(t, f, f.year.ID)
	request := startLeavingRequestForCancel(t, f)
	createLinkedCancelSettlement(t, f, request, account, 50_000)
	assertCancelBlockedWithoutMutation(t, f, request, "belum diserahkan")
}

func testCancelPartialHandover(t *testing.T, f leavingStartFixture) {
	account := createCancelAccount(t, f, f.year.ID)
	request := startLeavingRequestForCancel(t, f)
	settlement := createLinkedCancelSettlement(t, f, request, account, 50_000)
	insertCancelMovement(t, f, settlement, domain.SettlementCashHandover, 20_000)
	assertCancelBlockedWithoutMutation(t, f, request, "penyerahan sebagian")
}

func assertCancelBlockedWithoutMutation(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest, reason string) {
	t.Helper()
	before := cancelSnapshot(t, f.db)
	if _, err := f.svc.Cancel(context.Background(), cancelInput(f, request.ID, reason)); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("Cancel() error=%v, want ErrInvalidTransition", err)
	}
	if !reflect.DeepEqual(before, cancelSnapshot(t, f.db)) {
		t.Fatal("rejected cancellation changed request, student, placement, finance, or audit")
	}
}

func testCancelExcessHandover(t *testing.T, f leavingStartFixture) {
	account := createCancelAccount(t, f, f.year.ID)
	request := startLeavingRequestForCancel(t, f)
	settlement := createLinkedCancelSettlement(t, f, request, account, 50_000)
	insertCancelMovement(t, f, settlement, domain.SettlementCashHandover, 60_000)
	before := cancelSnapshot(t, f.db)
	if _, err := f.svc.Cancel(context.Background(), cancelInput(f, request.ID, "kelebihan")); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("Cancel() error=%v, want ErrInvalidTransition", err)
	}
	if !reflect.DeepEqual(before, cancelSnapshot(t, f.db)) {
		t.Fatal("over-reconciled cancellation changed state")
	}
}

func testCancelInvalidFinancialLink(t *testing.T, f leavingStartFixture) {
	request := startLeavingRequestForCancel(t, f)
	foreignStudent := newServiceScopeStudent(t, "Siswa relasi asing "+uuid.NewString())
	if err := f.repos.Students.Create(context.Background(), foreignStudent); err != nil {
		t.Fatal(err)
	}
	foreignAccount := createCancelAccountForStudent(t, f, foreignStudent.ID, f.year.ID)
	settlement, err := domain.NewSavingsSettlement(foreignAccount.ID, domain.SettlementStudentLeaving, 0, f.in.Actor.UserID, time.Now().UTC(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`INSERT INTO savings_settlements (id,savings_account_id,type,amount,executed_by,executed_at,status,student_leaving_request_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, settlement.ID, settlement.SavingsAccountID, settlement.Type, settlement.Amount, settlement.ExecutedBy, settlement.ExecutedAt, settlement.Status, request.ID); err != nil {
		t.Fatalf("seed malformed but FK-valid link: %v", err)
	}
	before := cancelSnapshot(t, f.db)
	if _, err := f.svc.Cancel(context.Background(), cancelInput(f, request.ID, "relasi tidak konsisten")); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("Cancel() error=%v, want ErrInvalidTransition", err)
	}
	if !reflect.DeepEqual(before, cancelSnapshot(t, f.db)) {
		t.Fatal("invalid link cancellation changed state")
	}
}

func testCancelFinalized(t *testing.T, f leavingStartFixture) {
	request := startLeavingRequestForCancel(t, f)
	if err := request.Finalize(f.in.Actor.UserID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := f.repos.StudentLeavingRequests.Update(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	before := cancelSnapshot(t, f.db)
	if _, err := f.svc.Cancel(context.Background(), cancelInput(f, request.ID, "sudah final")); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("Cancel() error=%v, want ErrInvalidTransition", err)
	}
	if !reflect.DeepEqual(before, cancelSnapshot(t, f.db)) {
		t.Fatal("finalized cancellation changed state")
	}
}

func testCancelRepeated(t *testing.T, f leavingStartFixture) {
	request := startLeavingRequestForCancel(t, f)
	if _, err := f.svc.Cancel(context.Background(), cancelInput(f, request.ID, "batal")); err != nil {
		t.Fatal(err)
	}
	before := cancelSnapshot(t, f.db)
	if _, err := f.svc.Cancel(context.Background(), cancelInput(f, request.ID, "batal lagi")); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("second Cancel() error=%v, want ErrInvalidTransition", err)
	}
	if !reflect.DeepEqual(before, cancelSnapshot(t, f.db)) {
		t.Fatal("repeated cancellation changed state")
	}
}

func testCancelUnauthorized(t *testing.T, f leavingStartFixture) {
	request := startLeavingRequestForCancel(t, f)
	before := cancelSnapshot(t, f.db)
	for _, role := range []domain.UserRole{domain.RoleWaliKelas, domain.RoleSiswa} {
		input := cancelInput(f, request.ID, "tidak berwenang")
		input.Actor.Role = role
		if _, err := f.svc.Cancel(context.Background(), input); !errors.Is(err, service.ErrForbidden) {
			t.Fatalf("Cancel(%s) error=%v, want ErrForbidden", role, err)
		}
	}
	if !reflect.DeepEqual(before, cancelSnapshot(t, f.db)) {
		t.Fatal("unauthorized cancellation changed state")
	}
}

func testCancelInvalidReason(t *testing.T, f leavingStartFixture) {
	request := startLeavingRequestForCancel(t, f)
	before := cancelSnapshot(t, f.db)
	for _, reason := range []string{"  ", string(make([]rune, 501))} {
		input := cancelInput(f, request.ID, reason)
		if _, err := f.svc.Cancel(context.Background(), input); !errors.Is(err, domain.ErrInvalidValue) {
			t.Fatalf("Cancel(reason length=%d) error=%v, want ErrInvalidValue", len([]rune(reason)), err)
		}
	}
	if !reflect.DeepEqual(before, cancelSnapshot(t, f.db)) {
		t.Fatal("invalid reason changed state")
	}
}

func testCancelMissingRequest(t *testing.T, f leavingStartFixture) {
	before := cancelSnapshot(t, f.db)
	if _, err := f.svc.Cancel(context.Background(), cancelInput(f, uuid.New(), "id tidak ada")); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("Cancel(missing) error=%v, want ErrNotFound", err)
	}
	if !reflect.DeepEqual(before, cancelSnapshot(t, f.db)) {
		t.Fatal("missing request changed another student's state")
	}
}

func testCancelOtherRequestIsolation(t *testing.T, f leavingStartFixture) {
	first := startLeavingRequestForCancel(t, f)
	otherStudent := f.addStudent(t)
	otherInput := f.in
	otherInput.StudentID = otherStudent.ID
	other, err := f.svc.Start(context.Background(), otherInput)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Cancel(context.Background(), cancelInput(f, first.ID, "isolasi proses")); err != nil {
		t.Fatal(err)
	}
	got, err := f.repos.StudentLeavingRequests.GetByID(context.Background(), other.Request.ID)
	if err != nil || got.Status != domain.StudentLeavingInProgress {
		t.Fatalf("proses siswa lain status=%s err=%v", got.Status, err)
	}
	student, err := f.repos.Students.GetByID(context.Background(), otherStudent.ID)
	if err != nil || !student.IsActive() {
		t.Fatalf("siswa lain status=%s err=%v", student.Status, err)
	}
}

func testCancelHistoricalNullSettlement(t *testing.T, f leavingStartFixture) {
	account := createCancelAccount(t, f, f.year.ID)
	settleCancelAccount(t, f, account)
	historical, err := domain.NewSavingsSettlement(account.ID, domain.SettlementRegularYearEnd, 0, f.in.Actor.UserID, time.Now().UTC(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.Settlements.Create(context.Background(), historical); err != nil {
		t.Fatal(err)
	}
	request := startLeavingRequestForCancel(t, f)
	before := cancelSnapshot(t, f.db)
	if _, err := f.svc.Cancel(context.Background(), cancelInput(f, request.ID, "histori tetap")); err != nil {
		t.Fatal(err)
	}
	got, err := f.repos.Settlements.GetByID(context.Background(), historical.ID)
	if err != nil || got.Status != domain.SettlementCompleted || got.Type != domain.SettlementRegularYearEnd {
		t.Fatalf("historical NULL-linked settlement changed: %+v, %v", got, err)
	}
	assertCancellationPreservedNonLifecycleData(t, before, cancelSnapshot(t, f.db))
}

func testCancelAccountsAcrossYears(t *testing.T, f leavingStartFixture) {
	ctx := context.Background()
	// Bentuk rekening historis di tahun tertutup sebelum membuka tahun referensi proses.
	currentYear := f.year
	currentYear.Status = domain.AcademicYearClosed
	currentYear.UpdatedAt = time.Now().UTC()
	if err := f.repos.AcademicYears.Update(ctx, currentYear); err != nil {
		t.Fatal(err)
	}
	oldYear := newServiceScopeAcademicYearForStartYear(t, "Tahun historis cancel "+uuid.NewString(), currentYear.StartDate.Year()-1)
	if err := f.repos.AcademicYears.Create(ctx, oldYear); err != nil {
		t.Fatal(err)
	}
	oldAccount := createCancelAccount(t, f, oldYear.ID)
	settleCancelAccount(t, f, oldAccount)
	historical, err := domain.NewSavingsSettlement(oldAccount.ID, domain.SettlementRegularYearEnd, 0, f.in.Actor.UserID, time.Now().UTC(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.Settlements.Create(ctx, historical); err != nil {
		t.Fatal(err)
	}
	oldYear.Status = domain.AcademicYearClosed
	oldYear.UpdatedAt = time.Now().UTC()
	if err := f.repos.AcademicYears.Update(ctx, oldYear); err != nil {
		t.Fatal(err)
	}
	currentYear.Status = domain.AcademicYearOpen
	currentYear.UpdatedAt = time.Now().UTC()
	if err := f.repos.AcademicYears.Update(ctx, currentYear); err != nil {
		t.Fatal(err)
	}
	newAccount := createCancelAccount(t, f, currentYear.ID)
	request := startLeavingRequestForCancel(t, f)
	before := cancelSnapshot(t, f.db)
	lockedYears := []uuid.UUID{}
	manager := cancelYearOrderManager{UnitOfWorkManager: f.manager, locked: &lockedYears}
	cancelSvc, err := service.NewStudentLeavingRequestService(service.Dependencies{Repositories: f.repos, UOW: manager})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cancelSvc.Cancel(ctx, cancelInput(f, request.ID, "lintas tahun")); err != nil {
		t.Fatal(err)
	}
	accounts, err := f.repos.SavingsAccounts.ListByStudent(ctx, f.in.StudentID)
	if err != nil {
		t.Fatal(err)
	}
	wantYears := []uuid.UUID{request.AcademicYearID}
	for _, account := range accounts {
		found := false
		for _, yearID := range wantYears {
			if yearID == account.AcademicYearID {
				found = true
				break
			}
		}
		if !found {
			wantYears = append(wantYears, account.AcademicYearID)
		}
	}
	sort.Slice(wantYears, func(i, j int) bool { return wantYears[i].String() < wantYears[j].String() })
	if !reflect.DeepEqual(lockedYears, wantYears) {
		t.Fatalf("tahun yang dikunci=%v, ingin urutan deterministik %v", lockedYears, wantYears)
	}
	for _, id := range []uuid.UUID{oldAccount.ID, newAccount.ID} {
		if _, err := f.repos.SavingsAccounts.GetByID(ctx, id); err != nil {
			t.Fatalf("account %s changed/disappeared: %v", id, err)
		}
	}
	assertCancellationPreservedNonLifecycleData(t, before, cancelSnapshot(t, f.db))
}

type cancelYearOrderManager struct {
	repository.UnitOfWorkManager
	locked *[]uuid.UUID
}

func (m cancelYearOrderManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	uow, err := m.UnitOfWorkManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := uow.Repositories()
	repos.AcademicYears = cancelYearOrderRepository{AcademicYearRepository: repos.AcademicYears, locked: m.locked}
	return cancelPauseUOW{UnitOfWork: uow, repos: repos}, nil
}

type cancelYearOrderRepository struct {
	repository.AcademicYearRepository
	locked *[]uuid.UUID
}

func (r cancelYearOrderRepository) GetByIDForShare(ctx context.Context, id uuid.UUID) (domain.AcademicYear, error) {
	year, err := r.AcademicYearRepository.GetByIDForShare(ctx, id)
	if err == nil {
		*r.locked = append(*r.locked, id)
	}
	return year, err
}

func createCancelAccount(t *testing.T, f leavingStartFixture, yearID uuid.UUID) domain.SavingsAccount {
	t.Helper()
	return createCancelAccountForStudent(t, f, f.in.StudentID, yearID)
}

func createCancelAccountForStudent(t *testing.T, f leavingStartFixture, studentID, yearID uuid.UUID) domain.SavingsAccount {
	t.Helper()
	account, err := domain.NewSavingsAccount(studentID, yearID)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.SavingsAccounts.Create(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	return account
}

func startLeavingRequestForCancel(t *testing.T, f leavingStartFixture) domain.StudentLeavingRequest {
	t.Helper()
	out, err := f.svc.Start(context.Background(), f.in)
	if err != nil || out.Request == nil {
		t.Fatalf("Start()=%+v, %v", out, err)
	}
	return *out.Request
}

func cancelInput(f leavingStartFixture, requestID uuid.UUID, reason string) service.CancelStudentLeavingRequestInput {
	return service.CancelStudentLeavingRequestInput{Actor: f.in.Actor, StudentLeavingRequestID: requestID, Reason: reason}
}

func createLinkedCancelSettlement(t *testing.T, f leavingStartFixture, request domain.StudentLeavingRequest, account domain.SavingsAccount, amount domain.Money) domain.SavingsSettlement {
	t.Helper()
	settleCancelAccount(t, f, account)
	settlement, err := domain.NewSavingsSettlement(account.ID, domain.SettlementStudentLeaving, amount, f.in.Actor.UserID, time.Now().UTC(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.Settlements.CreateForStudentLeavingRequest(context.Background(), settlement, request.ID); err != nil {
		t.Fatal(err)
	}
	return settlement
}

func settleCancelAccount(t *testing.T, f leavingStartFixture, account domain.SavingsAccount) {
	t.Helper()
	locked, err := f.repos.SavingsAccounts.GetByID(context.Background(), account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := locked.Settle(time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := f.repos.SavingsAccounts.Update(context.Background(), locked); err != nil {
		t.Fatal(err)
	}
}

func addCancelWali(t *testing.T, f leavingStartFixture) (domain.User, service.Actor) {
	t.Helper()
	wali := newServiceScopeUser(t, domain.RoleWaliKelas)
	if err := f.repos.Users.Create(context.Background(), wali); err != nil {
		t.Fatal(err)
	}
	assignment := newServiceScopeAssignment(t, wali.ID, f.year.ID, f.class.ID)
	if err := f.repos.TeacherClassAssignments.Create(context.Background(), assignment); err != nil {
		t.Fatal(err)
	}
	return wali, service.Actor{UserID: wali.ID, Role: domain.RoleWaliKelas}
}

func mustCancelDeposit(t *testing.T, f leavingStartFixture, wali domain.User, account domain.SavingsAccount, amount domain.Money) domain.Transaction {
	t.Helper()
	txService, err := service.NewTransactionService(service.Dependencies{Repositories: f.repos, UOW: f.manager})
	if err != nil {
		t.Fatal(err)
	}
	out, err := txService.Deposit(context.Background(), service.DepositInput{Actor: service.Actor{UserID: wali.ID, Role: domain.RoleWaliKelas}, SavingsAccountID: account.ID, Amount: amount, TransactionDate: f.in.EffectiveLeavingDate})
	if err != nil {
		t.Fatal(err)
	}
	return *out.Transaction
}

func insertCancelMovement(t *testing.T, f leavingStartFixture, settlement domain.SavingsSettlement, kind domain.SettlementCashMovementKind, amount domain.Money) {
	t.Helper()
	movement, err := domain.NewSettlementCashMovement(settlement.ID, kind, amount, f.in.Actor.UserID, time.Now().UTC(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repos.Settlements.CreateCashMovement(context.Background(), movement); err != nil {
		t.Fatal(err)
	}
}

func cancelSnapshot(t *testing.T, db *sql.DB) map[string]string {
	t.Helper()
	snapshot := make(map[string]string)
	for _, table := range []string{"students", "academic_years", "student_class_histories", "savings_accounts", "transactions", "savings_settlements", "settlement_cash_movements", "student_leaving_requests", "audit_logs"} {
		var value string
		query := fmt.Sprintf(`SELECT COALESCE(jsonb_agg(to_jsonb(row_data) ORDER BY row_data.id), '[]'::jsonb)::text FROM %s row_data`, table)
		if err := db.QueryRow(query).Scan(&value); err != nil {
			t.Fatal(err)
		}
		snapshot[table] = value
	}
	return snapshot
}

func assertCancelledAudit(t *testing.T, db *sql.DB, request domain.StudentLeavingRequest, actorID uuid.UUID) {
	t.Helper()
	var count int
	var payloadJSON string
	if err := db.QueryRow(`SELECT count(*), max(after_data::text) FROM audit_logs WHERE action='STUDENT_LEAVING_CANCELLED' AND entity_id=$1`, request.StudentID).Scan(&count, &payloadJSON); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("cancellation audit count=%d, want 1", count)
	}
	var payload struct {
		RequestID uuid.UUID                          `json:"student_leaving_request_id"`
		StudentID uuid.UUID                          `json:"student_id"`
		YearID    uuid.UUID                          `json:"academic_year_id"`
		Reason    string                             `json:"reason"`
		Status    domain.StudentLeavingRequestStatus `json:"status"`
		ActorID   uuid.UUID                          `json:"actor_id"`
	}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.RequestID != request.ID || payload.StudentID != request.StudentID || payload.YearID != request.AcademicYearID || payload.Reason != *request.CancellationReason || payload.Status != domain.StudentLeavingCancelled || payload.ActorID != actorID {
		t.Fatalf("cancellation audit payload=%+v", payload)
	}
}

func assertCancellationPreservedNonLifecycleData(t *testing.T, before, after map[string]string) {
	t.Helper()
	for _, table := range []string{"students", "academic_years", "student_class_histories", "savings_accounts", "transactions", "savings_settlements", "settlement_cash_movements"} {
		if before[table] != after[table] {
			t.Fatalf("Cancel changed %s: before=%s after=%s", table, before[table], after[table])
		}
	}
}

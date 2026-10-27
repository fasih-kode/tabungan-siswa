package service

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type cancelUnitRequestRepository struct {
	repository.StudentLeavingRequestRepository
	request domain.StudentLeavingRequest
	calls   *[]string
	err     error
}

func (r *cancelUnitRequestRepository) GetByID(_ context.Context, id uuid.UUID) (domain.StudentLeavingRequest, error) {
	if r.calls != nil {
		*r.calls = append(*r.calls, "request-pre-read")
	}
	if r.request.ID != id {
		return domain.StudentLeavingRequest{}, repository.ErrNotFound
	}
	return r.request, nil
}

func (r *cancelUnitRequestRepository) GetByIDForUpdate(_ context.Context, id uuid.UUID) (domain.StudentLeavingRequest, error) {
	*r.calls = append(*r.calls, "request-lock")
	if r.request.ID != id {
		return domain.StudentLeavingRequest{}, repository.ErrNotFound
	}
	return r.request, nil
}

func (r *cancelUnitRequestRepository) ListFinancialLinkIssues(context.Context, uuid.UUID) ([]repository.StudentLeavingFinancialLinkIssue, error) {
	*r.calls = append(*r.calls, "integrity")
	return nil, nil
}

func (r *cancelUnitRequestRepository) Update(_ context.Context, request domain.StudentLeavingRequest) error {
	*r.calls = append(*r.calls, "request-update")
	if r.err != nil {
		return r.err
	}
	r.request = request
	return nil
}

type cancelUnitSavingsAccountRepository struct {
	repository.SavingsAccountRepository
	accounts []domain.SavingsAccount
	calls    *[]string
}

func (r *cancelUnitSavingsAccountRepository) ListByStudent(context.Context, uuid.UUID) ([]domain.SavingsAccount, error) {
	if r.calls != nil {
		*r.calls = append(*r.calls, "accounts-list")
	}
	return append([]domain.SavingsAccount(nil), r.accounts...), nil
}

func (r *cancelUnitSavingsAccountRepository) ListByStudentForUpdate(context.Context, uuid.UUID) ([]domain.SavingsAccount, error) {
	*r.calls = append(*r.calls, "accounts-lock")
	return append([]domain.SavingsAccount(nil), r.accounts...), nil
}

type cancelUnitAcademicYearRepository struct {
	repository.AcademicYearRepository
	years map[uuid.UUID]domain.AcademicYear
	calls *[]string
}

func (r cancelUnitAcademicYearRepository) GetByIDForShare(_ context.Context, id uuid.UUID) (domain.AcademicYear, error) {
	*r.calls = append(*r.calls, "year-lock:"+id.String())
	return r.years[id], nil
}

type cancelUnitStudentRepository struct {
	repository.StudentRepository
	student domain.Student
	calls   *[]string
}

func (r cancelUnitStudentRepository) GetByIDForUpdate(context.Context, uuid.UUID) (domain.Student, error) {
	*r.calls = append(*r.calls, "student-lock")
	return r.student, nil
}

type cancelUnitSettlementRepository struct {
	repository.SettlementRepository
	settlements []domain.SavingsSettlement
	movements   map[uuid.UUID][]domain.SettlementCashMovement
	calls       *[]string
}

func (r cancelUnitSettlementRepository) ListByStudentLeavingRequest(context.Context, uuid.UUID) ([]domain.SavingsSettlement, error) {
	*r.calls = append(*r.calls, "settlements")
	return r.settlements, nil
}

func (r cancelUnitSettlementRepository) ListCashMovements(_ context.Context, id uuid.UUID) ([]domain.SettlementCashMovement, error) {
	*r.calls = append(*r.calls, "cash-movements")
	return r.movements[id], nil
}

type cancelUnitAuditRepository struct {
	repository.AuditRepository
	audits []domain.AuditLog
	calls  *[]string
	err    error
}

func (r *cancelUnitAuditRepository) Create(_ context.Context, audit domain.AuditLog) error {
	*r.calls = append(*r.calls, "audit")
	if r.err != nil {
		return r.err
	}
	r.audits = append(r.audits, audit)
	return nil
}

func TestStudentLeavingCancelUnitLockOrderAndAudit(t *testing.T) {
	f := newCancelUnitFixture(t)
	output, err := f.service.Cancel(context.Background(), f.input)
	if err != nil {
		t.Fatal(err)
	}
	if output.Request == nil || output.Request.Status != domain.StudentLeavingCancelled ||
		output.Request.CancelledBy == nil || *output.Request.CancelledBy != f.input.Actor.UserID ||
		output.Request.CancellationReason == nil || *output.Request.CancellationReason != "dibatalkan" {
		t.Fatalf("Cancel() output=%+v", output)
	}
	orderedYears := studentLeavingYearIDs(f.request.AcademicYearID, f.accounts.accounts)
	want := []string{"request-pre-read", "accounts-list", "year-lock:" + orderedYears[0].String(), "year-lock:" + orderedYears[1].String(), "accounts-lock", "student-lock", "request-lock", "accounts-list", "integrity", "settlements", "request-update", "audit"}
	if !reflect.DeepEqual(f.calls, want) {
		t.Fatalf("call order=%v, want %v", f.calls, want)
	}
	if len(f.audits.audits) != 1 {
		t.Fatalf("audit count=%d, want 1", len(f.audits.audits))
	}
	audit := f.audits.audits[0]
	if audit.Action != "STUDENT_LEAVING_CANCELLED" || audit.EntityType != domain.AuditEntityStudent || audit.EntityID != f.request.StudentID {
		t.Fatalf("cancellation audit=%+v", audit)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(audit.AfterData, &payload); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"student_leaving_request_id", "student_id", "academic_year_id", "reason", "status", "actor_id", "actor_role", "occurred_at"} {
		if len(payload[key]) == 0 {
			t.Fatalf("cancellation audit is missing %q", key)
		}
	}
	if f.uow.commitCount != 1 || f.uow.rollbackCount != 0 {
		t.Fatalf("commit/rollback=%d/%d", f.uow.commitCount, f.uow.rollbackCount)
	}
}

func TestStudentLeavingCancelUnitRejectsUnauthorizedActorBeforeTransaction(t *testing.T) {
	f := newCancelUnitFixture(t)
	f.input.Actor.Role = domain.RoleWaliKelas
	if _, err := f.service.Cancel(context.Background(), f.input); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Cancel() error=%v, want ErrForbidden", err)
	}
	if f.manager.beginCount != 0 {
		t.Fatalf("UOW Begin calls=%d, want 0", f.manager.beginCount)
	}
}

func TestStudentLeavingCancellationHandoverValidation(t *testing.T) {
	settlementID, actorID := uuid.New(), uuid.New()
	confirmedAt := time.Now().UTC()
	validHandover := domain.SettlementCashMovement{ID: uuid.New(), SavingsSettlementID: settlementID, Kind: domain.SettlementCashHandover, Amount: 80, ConfirmedBy: actorID, ConfirmedAt: confirmedAt}
	validReturn := domain.SettlementCashMovement{ID: uuid.New(), SavingsSettlementID: settlementID, Kind: domain.SettlementCashReturn, Amount: 20, ConfirmedBy: actorID, ConfirmedAt: confirmedAt}
	cases := []struct {
		name      string
		amount    domain.Money
		movements []domain.SettlementCashMovement
		wantErr   bool
	}{
		{name: "nol settlement tanpa handover", amount: 0},
		{name: "penyerahan lengkap", amount: 60, movements: []domain.SettlementCashMovement{validHandover, validReturn}},
		{name: "penyerahan belum dilakukan", amount: 60, wantErr: true},
		{name: "penyerahan sebagian", amount: 60, movements: []domain.SettlementCashMovement{{ID: uuid.New(), SavingsSettlementID: settlementID, Kind: domain.SettlementCashHandover, Amount: 30, ConfirmedBy: actorID, ConfirmedAt: confirmedAt}}, wantErr: true},
		{name: "penyerahan berlebih", amount: 60, movements: []domain.SettlementCashMovement{{ID: uuid.New(), SavingsSettlementID: settlementID, Kind: domain.SettlementCashHandover, Amount: 80, ConfirmedBy: actorID, ConfirmedAt: confirmedAt}}, wantErr: true},
		{name: "saldo bersih negatif", amount: 0, movements: []domain.SettlementCashMovement{validReturn}, wantErr: true},
		{name: "jenis movement tidak dikenal", amount: 0, movements: []domain.SettlementCashMovement{{ID: uuid.New(), SavingsSettlementID: settlementID, Kind: "OTHER", Amount: 1, ConfirmedBy: actorID, ConfirmedAt: confirmedAt}}, wantErr: true},
		{name: "nominal movement tidak sah", amount: 0, movements: []domain.SettlementCashMovement{{ID: uuid.New(), SavingsSettlementID: settlementID, Kind: domain.SettlementCashHandover, Amount: -1, ConfirmedBy: actorID, ConfirmedAt: confirmedAt}}, wantErr: true},
		{name: "waktu konfirmasi tidak sah", amount: 0, movements: []domain.SettlementCashMovement{{ID: uuid.New(), SavingsSettlementID: settlementID, Kind: domain.SettlementCashHandover, Amount: 1, ConfirmedBy: actorID}}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateStudentLeavingHandover(tc.amount, tc.movements)
			if tc.wantErr && !errors.Is(err, domain.ErrInvalidTransition) {
				t.Fatalf("validation error=%v, want ErrInvalidTransition", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("validation error=%v", err)
			}
		})
	}
}

type cancelUnitFixture struct {
	service  *studentLeavingRequestService
	input    CancelStudentLeavingRequestInput
	request  domain.StudentLeavingRequest
	manager  *studentServiceFakeUnitOfWorkManager
	uow      *studentServiceFakeUnitOfWork
	audits   *cancelUnitAuditRepository
	accounts *cancelUnitSavingsAccountRepository
	calls    []string
}

func newCancelUnitFixture(t *testing.T) *cancelUnitFixture {
	t.Helper()
	f := &cancelUnitFixture{calls: []string{}}
	studentID, yearID, requestID, adminID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	f.request = domain.StudentLeavingRequest{ID: requestID, StudentID: studentID, AcademicYearID: yearID, EffectiveLeavingDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Status: domain.StudentLeavingInProgress, InitiatedBy: adminID, CreatedAt: time.Now().UTC()}
	f.input = CancelStudentLeavingRequestInput{Actor: Actor{UserID: adminID, Role: domain.RoleAdmin}, StudentLeavingRequestID: requestID, Reason: "  dibatalkan  "}
	otherYearID := uuid.New()
	year := domain.AcademicYear{ID: yearID, Status: domain.AcademicYearClosed}
	otherYear := domain.AcademicYear{ID: otherYearID, Status: domain.AcademicYearOpen}
	requestRepo := &cancelUnitRequestRepository{request: f.request, calls: &f.calls}
	firstAccountID, secondAccountID := uuid.New(), uuid.New()
	if firstAccountID.String() < secondAccountID.String() {
		firstAccountID, secondAccountID = secondAccountID, firstAccountID
	}
	savingsRepo := &cancelUnitSavingsAccountRepository{calls: &f.calls, accounts: []domain.SavingsAccount{
		{ID: firstAccountID, StudentID: studentID, AcademicYearID: yearID, Status: domain.SavingsAccountOpen},
		{ID: secondAccountID, StudentID: studentID, AcademicYearID: otherYearID, Status: domain.SavingsAccountOpen},
	}}
	f.accounts = savingsRepo
	f.audits = &cancelUnitAuditRepository{calls: &f.calls}
	repos := repository.RepositorySet{
		AcademicYears:          cancelUnitAcademicYearRepository{years: map[uuid.UUID]domain.AcademicYear{yearID: year, otherYearID: otherYear}, calls: &f.calls},
		SavingsAccounts:        savingsRepo,
		Students:               cancelUnitStudentRepository{student: domain.Student{ID: studentID, Status: domain.StudentActive}, calls: &f.calls},
		StudentLeavingRequests: requestRepo,
		Settlements:            cancelUnitSettlementRepository{calls: &f.calls},
		AuditLogs:              f.audits,
	}
	// Pembacaan ulang setelah lock siswa harus menunjukkan himpunan yang sama.
	manager := &studentServiceFakeUnitOfWorkManager{}
	f.uow = &studentServiceFakeUnitOfWork{repos: repos}
	manager.uow = f.uow
	f.manager = manager
	f.service = &studentLeavingRequestService{deps: Dependencies{Repositories: repos, UOW: manager}, now: func() time.Time { return time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC) }}
	return f
}

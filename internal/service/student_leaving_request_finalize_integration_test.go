package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/fasih/tabungan-siswa/internal/service"
	"github.com/google/uuid"
)

func assertFinalizeMatchesReadiness(t *testing.T, f leavingStartFixture, id uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	ready, err := f.svc.Readiness(ctx, service.StudentLeavingReadinessInput{Actor: f.in.Actor, StudentLeavingRequestID: id})
	if err != nil {
		t.Fatal(err)
	}
	before := readinessSnapshot(t, f.db)
	out, err := f.svc.Finalize(ctx, service.FinalizeStudentLeavingRequestInput{Actor: f.in.Actor, StudentLeavingRequestID: id})
	if !ready.Ready {
		if !errors.Is(err, domain.ErrInvalidTransition) || !reflect.DeepEqual(before, readinessSnapshot(t, f.db)) {
			t.Fatalf("Finalize melewati blocker/bermutasi: %v %+v", err, ready.Blockers)
		}
		return
	}
	if err != nil || out.Student == nil || out.Request == nil {
		t.Fatalf("Finalize()=%+v,%v", out, err)
	}
	student, err := f.repos.Students.GetByID(ctx, ready.Student.ID)
	if err != nil {
		t.Fatal(err)
	}
	request, err := f.repos.StudentLeavingRequests.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if !student.IsLeft() || student.LeavingDate == nil || !student.LeavingDate.Equal(ready.Request.EffectiveLeavingDate) || request.Status != domain.StudentLeavingFinalized || request.FinalizedBy == nil || *request.FinalizedBy != f.in.Actor.UserID || request.FinalizedAt == nil || request.FinalizedAt.Location() != time.UTC {
		t.Fatalf("hasil lifecycle tidak konsisten: %+v %+v", student, request)
	}
	if _, err := f.repos.StudentClassHistories.GetCurrentByStudentAndAcademicYear(ctx, student.ID, request.AcademicYearID, request.EffectiveLeavingDate); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("placement belum ditutup: %v", err)
	}
	after := readinessSnapshot(t, f.db)
	for table, value := range before {
		if table != "students" && table != "student_class_histories" && table != "student_leaving_requests" && table != "audit_logs" && after[table] != value {
			t.Fatalf("Finalize mengubah %s", table)
		}
	}
	var count int
	var payload []byte
	if err := f.db.QueryRow(`SELECT count(*) FROM audit_logs WHERE entity_id=$1 AND action='STUDENT_LEAVING_FINALIZED'`, student.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("audit=%d %v", count, err)
	}
	if err := f.db.QueryRow(`SELECT after_data FROM audit_logs WHERE entity_id=$1 AND action='STUDENT_LEAVING_FINALIZED'`, student.ID).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	var audit struct {
		RequestID            uuid.UUID                          `json:"request_id"`
		StudentID            uuid.UUID                          `json:"student_id"`
		AcademicYearID       uuid.UUID                          `json:"academic_year_id"`
		EffectiveLeavingDate string                             `json:"effective_leaving_date"`
		FinalizedAt          time.Time                          `json:"finalized_at"`
		ActorID              uuid.UUID                          `json:"actor_id"`
		ActorRole            domain.UserRole                    `json:"actor_role"`
		RequestStatus        domain.StudentLeavingRequestStatus `json:"request_status"`
		StudentStatus        domain.StudentStatus               `json:"student_status"`
	}
	if err := json.Unmarshal(payload, &audit); err != nil {
		t.Fatal(err)
	}
	if audit.RequestID != id || audit.StudentID != student.ID || audit.AcademicYearID != request.AcademicYearID || audit.EffectiveLeavingDate != request.EffectiveLeavingDate.Format("2006-01-02") || !audit.FinalizedAt.Equal(*request.FinalizedAt) || audit.ActorID != f.in.Actor.UserID || audit.ActorRole != domain.RoleAdmin || audit.RequestStatus != domain.StudentLeavingFinalized || audit.StudentStatus != domain.StudentLeft {
		t.Fatalf("audit=%+v", audit)
	}
	if _, err := f.svc.Finalize(ctx, service.FinalizeStudentLeavingRequestInput{Actor: f.in.Actor, StudentLeavingRequestID: id}); !errors.Is(err, domain.ErrInvalidTransition) || !reflect.DeepEqual(after, readinessSnapshot(t, f.db)) {
		t.Fatalf("finalisasi ulang: %v", err)
	}
}

func TestStudentLeavingFinalizePlacement(t *testing.T) {
	for _, mode := range []string{"tanggal mendatang", "awal placement", "placement lebih baru", "placement hilang", "transfer pending", "role Wali", "role Siswa"} {
		t.Run(mode, func(t *testing.T) {
			f := newLeavingCancelFixture(t)
			if mode == "awal placement" {
				f.in.EffectiveLeavingDate = f.year.StartDate
			}
			request := startLeavingRequestForCancel(t, f)
			if mode == "tanggal mendatang" {
				if _, err := f.db.Exec(`UPDATE student_leaving_requests SET effective_leaving_date=$2 WHERE id=$1`, request.ID, f.in.EffectiveLeavingDate.AddDate(0, 0, 1)); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "placement lebih baru" {
				date := request.EffectiveLeavingDate.AddDate(0, 0, 1)
				if _, err := f.db.Exec(`UPDATE student_class_histories SET validity=daterange(lower(validity),$2::date,'[)') WHERE student_id=$1`, f.in.StudentID, date); err != nil {
					t.Fatal(err)
				}
				period, err := domain.NewDateRange(date, &f.year.EndDate)
				if err != nil {
					t.Fatal(err)
				}
				history, err := domain.NewStudentClassHistory(f.in.StudentID, f.year.ID, f.class.ID, period)
				if err != nil {
					t.Fatal(err)
				}
				if err := f.repos.StudentClassHistories.Create(context.Background(), history); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "placement hilang" {
				if _, err := f.db.Exec(`UPDATE student_class_histories SET validity=daterange(lower(validity),$2::date,'[)') WHERE student_id=$1`, f.in.StudentID, request.EffectiveLeavingDate); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "transfer pending" {
				target := newServiceScopeClass(t, "Kelas tujuan finalisasi", 8)
				target.AcademicYearID = f.year.ID
				target.EffectiveDate = f.year.StartDate
				if err := f.repos.Classes.Create(context.Background(), target); err != nil {
					t.Fatal(err)
				}
				transfer, err := domain.NewClassTransferRequest(f.in.StudentID, f.year.ID, f.class.ID, target.ID, f.in.Actor.UserID)
				if err != nil {
					t.Fatal(err)
				}
				if err := f.repos.ClassTransferRequests.Create(context.Background(), transfer); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "role Wali" || mode == "role Siswa" {
				actor := f.in.Actor
				actor.Role = domain.RoleWaliKelas
				if mode == "role Siswa" {
					actor.Role = domain.RoleSiswa
				}
				before := readinessSnapshot(t, f.db)
				if _, err := f.svc.Finalize(context.Background(), service.FinalizeStudentLeavingRequestInput{Actor: actor, StudentLeavingRequestID: request.ID}); !errors.Is(err, service.ErrForbidden) || !reflect.DeepEqual(before, readinessSnapshot(t, f.db)) {
					t.Fatalf("otorisasi: %v", err)
				}
			} else {
				out := readStudentLeavingReadiness(t, f, request.ID)
				if out.Ready {
					t.Fatal("placement/transfer seharusnya memblokir")
				}
				assertFinalizeMatchesReadiness(t, f, request.ID)
			}
		})
	}
}

func TestStudentLeavingFinalizeRollback(t *testing.T) {
	for _, stage := range []string{"settlement read", "placement", "student", "request update", "audit after insert", "commit"} {
		t.Run(stage, func(t *testing.T) {
			f := newLeavingCancelFixture(t)
			request := startLeavingRequestForCancel(t, f)
			injected := errors.New("kegagalan finalisasi untuk tes rollback")
			var manager repository.UnitOfWorkManager = cancelFailureManager{UnitOfWorkManager: f.manager, stage: stage, failure: injected}
			if stage == "student" {
				manager = failingStudentUpdateManager{UnitOfWorkManager: f.manager, failure: injected}
			}
			if stage == "placement" {
				manager = finalizePlacementFailureManager{UnitOfWorkManager: f.manager, failure: injected}
			}
			svc, err := service.NewStudentLeavingRequestService(service.Dependencies{Repositories: f.repos, UOW: manager})
			if err != nil {
				t.Fatal(err)
			}
			before := readinessSnapshot(t, f.db)
			if _, err := svc.Finalize(context.Background(), service.FinalizeStudentLeavingRequestInput{Actor: f.in.Actor, StudentLeavingRequestID: request.ID}); !errors.Is(err, injected) {
				t.Fatalf("stage %s error=%v", stage, err)
			}
			if !reflect.DeepEqual(before, readinessSnapshot(t, f.db)) {
				t.Fatalf("rollback %s tidak utuh", stage)
			}
		})
	}
}

type finalizePlacementFailureManager struct {
	repository.UnitOfWorkManager
	failure error
}

func (m finalizePlacementFailureManager) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	uow, err := m.UnitOfWorkManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	repos := uow.Repositories()
	repos.StudentClassHistories = finalizePlacementFailure{StudentClassHistoryRepository: repos.StudentClassHistories, failure: m.failure}
	return cancelPauseUOW{UnitOfWork: uow, repos: repos}, nil
}

type finalizePlacementFailure struct {
	repository.StudentClassHistoryRepository
	failure error
}

func (r finalizePlacementFailure) CloseCurrentByStudentAndAcademicYear(ctx context.Context, s, y uuid.UUID, d time.Time) (domain.StudentClassHistory, error) {
	h, err := r.StudentClassHistoryRepository.CloseCurrentByStudentAndAcademicYear(ctx, s, y, d)
	if err != nil {
		return h, err
	}
	return h, r.failure
}

func TestStudentLeavingFinalizeRejectsStalePreRead(t *testing.T) {
	for _, mode := range []string{"rekening berubah", "tanggal berubah"} {
		t.Run(mode, func(t *testing.T) {
			f := newLeavingCancelFixture(t)
			if mode == "rekening berubah" {
				createCancelAccount(t, f, f.year.ID)
			}
			request := startLeavingRequestForCancel(t, f)
			initial := f.repos
			if mode == "rekening berubah" {
				initial.SavingsAccounts = finalizeStaleAccounts{SavingsAccountRepository: f.repos.SavingsAccounts}
			} else {
				initial.StudentLeavingRequests = finalizeStaleRequest{StudentLeavingRequestRepository: f.repos.StudentLeavingRequests}
			}
			svc, err := service.NewStudentLeavingRequestService(service.Dependencies{Repositories: initial, UOW: f.manager})
			if err != nil {
				t.Fatal(err)
			}
			before := readinessSnapshot(t, f.db)
			_, err = svc.Finalize(context.Background(), service.FinalizeStudentLeavingRequestInput{Actor: f.in.Actor, StudentLeavingRequestID: request.ID})
			want := repository.ErrConflict
			if mode == "tanggal berubah" {
				want = domain.ErrInvalidTransition
			}
			if !errors.Is(err, want) || !reflect.DeepEqual(before, readinessSnapshot(t, f.db)) {
				t.Fatalf("stale %s error=%v", mode, err)
			}
		})
	}
}

type finalizeStaleAccounts struct {
	repository.SavingsAccountRepository
}

func (r finalizeStaleAccounts) ListByStudent(context.Context, uuid.UUID) ([]domain.SavingsAccount, error) {
	return []domain.SavingsAccount{}, nil
}

type finalizeStaleRequest struct {
	repository.StudentLeavingRequestRepository
}

func (r finalizeStaleRequest) GetByID(ctx context.Context, id uuid.UUID) (domain.StudentLeavingRequest, error) {
	request, err := r.StudentLeavingRequestRepository.GetByID(ctx, id)
	request.EffectiveLeavingDate = request.EffectiveLeavingDate.AddDate(0, 0, -1)
	return request, err
}

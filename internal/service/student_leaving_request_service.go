package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type studentLeavingRequestService struct {
	deps Dependencies
	now  func() time.Time
}

var _ StudentLeavingRequestService = (*studentLeavingRequestService)(nil)

// NewStudentLeavingRequestService membangun service internal tanpa mendaftarkan jalur pengguna.
func NewStudentLeavingRequestService(deps Dependencies) (StudentLeavingRequestService, error) {
	if err := deps.Validate(); err != nil {
		return nil, err
	}
	return &studentLeavingRequestService{deps: deps, now: time.Now}, nil
}

func (s *studentLeavingRequestService) Start(ctx context.Context, input StartStudentLeavingRequestInput) (StartStudentLeavingRequestOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return StartStudentLeavingRequestOutput{}, err
	}
	if input.StudentID == uuid.Nil || input.AcademicYearID == uuid.Nil {
		return StartStudentLeavingRequestOutput{}, domain.ErrInvalidID
	}
	if input.EffectiveLeavingDate.IsZero() {
		return StartStudentLeavingRequestOutput{}, domain.ErrInvalidValue
	}

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return StartStudentLeavingRequestOutput{}, fmt.Errorf("begin student leaving request: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()
	repos := uow.Repositories()
	if repos.AcademicYears == nil || repos.Students == nil || repos.StudentClassHistories == nil || repos.StudentLeavingRequests == nil || repos.AuditLogs == nil {
		return StartStudentLeavingRequestOutput{}, ErrInvalidDependency
	}

	// Tahun adalah gerbang lifecycle; siswa menyerialkan Start meskipun row proses belum ada.
	year, err := repos.AcademicYears.GetByIDForUpdate(ctx, input.AcademicYearID)
	if err != nil {
		return StartStudentLeavingRequestOutput{}, err
	}
	if !year.IsOpen() {
		return StartStudentLeavingRequestOutput{}, domain.ErrInvalidTransition
	}
	student, err := repos.Students.GetByIDForUpdate(ctx, input.StudentID)
	if err != nil {
		return StartStudentLeavingRequestOutput{}, err
	}
	if !student.IsActive() {
		return StartStudentLeavingRequestOutput{}, domain.ErrInvalidTransition
	}

	date := time.Date(input.EffectiveLeavingDate.Year(), input.EffectiveLeavingDate.Month(), input.EffectiveLeavingDate.Day(), 0, 0, 0, 0, time.UTC)
	now := s.now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	// Pembatasan sementara: backdate, tanggal mendatang, dan zona tanggal bisnis belum disetujui.
	if !date.Equal(today) || date.Before(year.StartDate) || !date.Before(year.EndDate) {
		return StartStudentLeavingRequestOutput{}, domain.ErrInvalidDateRange
	}
	history, err := repos.StudentClassHistories.GetCurrentByStudentAndAcademicYear(ctx, student.ID, year.ID, date)
	if err != nil {
		return StartStudentLeavingRequestOutput{}, err
	}
	if history.StudentID != student.ID || history.AcademicYearID != year.ID || !history.Contains(date) {
		return StartStudentLeavingRequestOutput{}, domain.ErrInvalidTransition
	}
	if err := history.ValidateWithinAcademicYear(year.StartDate, year.EndDate); err != nil {
		return StartStudentLeavingRequestOutput{}, err
	}

	if _, err := repos.StudentLeavingRequests.GetInProgressByStudent(ctx, student.ID); err == nil {
		return StartStudentLeavingRequestOutput{}, repository.ErrConflict
	} else if !errors.Is(err, repository.ErrNotFound) {
		return StartStudentLeavingRequestOutput{}, err
	}
	request, err := domain.NewStudentLeavingRequest(student.ID, year.ID, date, input.Actor.UserID)
	if err != nil {
		return StartStudentLeavingRequestOutput{}, err
	}
	if err := repos.StudentLeavingRequests.Create(ctx, request); err != nil {
		return StartStudentLeavingRequestOutput{}, err
	}
	payload, err := json.Marshal(struct {
		StudentID            uuid.UUID                          `json:"student_id"`
		LeavingRequestID     uuid.UUID                          `json:"leaving_request_id"`
		AcademicYearID       uuid.UUID                          `json:"academic_year_id"`
		EffectiveLeavingDate string                             `json:"effective_leaving_date"`
		Status               domain.StudentLeavingRequestStatus `json:"status"`
		ActorID              uuid.UUID                          `json:"actor_id"`
		ActorRole            domain.UserRole                    `json:"actor_role"`
		OccurredAt           time.Time                          `json:"occurred_at"`
	}{
		StudentID:            student.ID,
		LeavingRequestID:     request.ID,
		AcademicYearID:       year.ID,
		EffectiveLeavingDate: request.EffectiveLeavingDate.Format("2006-01-02"),
		Status:               request.Status,
		ActorID:              input.Actor.UserID,
		ActorRole:            input.Actor.Role,
		OccurredAt:           request.CreatedAt,
	})
	if err != nil {
		return StartStudentLeavingRequestOutput{}, err
	}
	if _, err := recordAudit(ctx, repos, RecordAuditInput{
		Actor:      input.Actor,
		Action:     "STUDENT_LEAVING_STARTED",
		EntityType: domain.AuditEntityStudent,
		EntityID:   student.ID,
		After:      payload,
	}); err != nil {
		return StartStudentLeavingRequestOutput{}, err
	}
	if err := uow.Commit(); err != nil {
		return StartStudentLeavingRequestOutput{}, fmt.Errorf("commit student leaving request: %w", err)
	}
	committed = true
	return StartStudentLeavingRequestOutput{Request: &request}, nil
}

func (s *studentLeavingRequestService) Get(ctx context.Context, input GetStudentLeavingRequestInput) (GetStudentLeavingRequestOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return GetStudentLeavingRequestOutput{}, err
	}
	if input.RequestID == uuid.Nil {
		return GetStudentLeavingRequestOutput{}, domain.ErrInvalidID
	}
	if s.deps.Repositories.StudentLeavingRequests == nil {
		return GetStudentLeavingRequestOutput{}, ErrInvalidDependency
	}
	request, err := s.deps.Repositories.StudentLeavingRequests.GetByID(ctx, input.RequestID)
	if err != nil {
		return GetStudentLeavingRequestOutput{}, err
	}
	return GetStudentLeavingRequestOutput{Request: &request}, nil
}

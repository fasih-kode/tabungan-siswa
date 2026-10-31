package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

// Finalize mengesahkan pindah dalam satu UOW sesudah seluruh gerbang lifecycle terkunci.
func (s *studentLeavingRequestService) Finalize(ctx context.Context, input FinalizeStudentLeavingRequestInput) (FinalizeStudentLeavingRequestOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return FinalizeStudentLeavingRequestOutput{}, err
	}
	if input.StudentLeavingRequestID == uuid.Nil {
		return FinalizeStudentLeavingRequestOutput{}, domain.ErrInvalidID
	}
	if s.deps.Repositories.StudentLeavingRequests == nil || s.deps.Repositories.SavingsAccounts == nil {
		return FinalizeStudentLeavingRequestOutput{}, ErrInvalidDependency
	}
	initial, err := s.deps.Repositories.StudentLeavingRequests.GetByID(ctx, input.StudentLeavingRequestID)
	if err != nil {
		return FinalizeStudentLeavingRequestOutput{}, err
	}
	accounts, err := s.deps.Repositories.SavingsAccounts.ListByStudent(ctx, initial.StudentID)
	if err != nil {
		return FinalizeStudentLeavingRequestOutput{}, err
	}
	accounts = orderedStudentLeavingAccounts(accounts)
	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return FinalizeStudentLeavingRequestOutput{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()
	repos := uow.Repositories()
	if repos.AcademicYears == nil || repos.SavingsAccounts == nil || repos.Students == nil || repos.StudentLeavingRequests == nil || repos.AuditLogs == nil || repos.StudentClassHistories == nil {
		return FinalizeStudentLeavingRequestOutput{}, ErrInvalidDependency
	}
	// Seluruh tahun dikunci lebih dahulu; tidak ada lock tahun tambahan setelah rekening atau siswa.
	for _, id := range studentLeavingYearIDs(initial.AcademicYearID, accounts) {
		if year, err := repos.AcademicYears.GetByIDForUpdate(ctx, id); err != nil {
			return FinalizeStudentLeavingRequestOutput{}, err
		} else if year.ID != id {
			return FinalizeStudentLeavingRequestOutput{}, repository.ErrConflict
		}
	}
	locked, err := repos.SavingsAccounts.ListByStudentForUpdate(ctx, initial.StudentID)
	if err != nil {
		return FinalizeStudentLeavingRequestOutput{}, err
	}
	if !sameStudentLeavingAccounts(accounts, orderedStudentLeavingAccounts(locked)) {
		return FinalizeStudentLeavingRequestOutput{}, repository.ErrConflict
	}
	student, err := repos.Students.GetByIDForUpdate(ctx, initial.StudentID)
	if err != nil {
		return FinalizeStudentLeavingRequestOutput{}, err
	}
	request, err := repos.StudentLeavingRequests.GetByIDForUpdate(ctx, initial.ID)
	if err != nil {
		return FinalizeStudentLeavingRequestOutput{}, err
	}
	if student.ID != initial.StudentID || request.ID != initial.ID ||
		request.StudentID != initial.StudentID || request.AcademicYearID != initial.AcademicYearID ||
		!request.EffectiveLeavingDate.Equal(initial.EffectiveLeavingDate) ||
		request.Status != domain.StudentLeavingInProgress || !student.IsActive() {
		return FinalizeStudentLeavingRequestOutput{}, domain.ErrInvalidTransition
	}
	current, err := repos.SavingsAccounts.ListByStudent(ctx, student.ID)
	if err != nil {
		return FinalizeStudentLeavingRequestOutput{}, err
	}
	if !sameStudentLeavingAccounts(accounts, orderedStudentLeavingAccounts(current)) {
		return FinalizeStudentLeavingRequestOutput{}, repository.ErrConflict
	}
	readiness, err := s.studentLeavingReadiness(ctx, repos, request, student)
	if err != nil {
		return FinalizeStudentLeavingRequestOutput{}, err
	}
	if !readiness.Ready {
		return FinalizeStudentLeavingRequestOutput{}, domain.ErrInvalidTransition
	}
	if _, err := repos.StudentClassHistories.CloseCurrentByStudentAndAcademicYear(ctx, student.ID, request.AcademicYearID, request.EffectiveLeavingDate); err != nil {
		return FinalizeStudentLeavingRequestOutput{}, err
	}
	if err := student.LeaveOn(request.EffectiveLeavingDate); err != nil {
		return FinalizeStudentLeavingRequestOutput{}, err
	}
	if err := repos.Students.Update(ctx, student); err != nil {
		return FinalizeStudentLeavingRequestOutput{}, err
	}
	if err := request.Finalize(input.Actor.UserID, s.now().UTC().Truncate(time.Microsecond)); err != nil {
		return FinalizeStudentLeavingRequestOutput{}, err
	}
	if err := repos.StudentLeavingRequests.Update(ctx, request); err != nil {
		return FinalizeStudentLeavingRequestOutput{}, err
	}
	payload, err := json.Marshal(struct {
		RequestID            uuid.UUID                          `json:"request_id"`
		StudentID            uuid.UUID                          `json:"student_id"`
		AcademicYearID       uuid.UUID                          `json:"academic_year_id"`
		EffectiveLeavingDate string                             `json:"effective_leaving_date"`
		FinalizedAt          time.Time                          `json:"finalized_at"`
		ActorID              uuid.UUID                          `json:"actor_id"`
		ActorRole            domain.UserRole                    `json:"actor_role"`
		RequestStatus        domain.StudentLeavingRequestStatus `json:"request_status"`
		StudentStatus        domain.StudentStatus               `json:"student_status"`
	}{request.ID, student.ID, request.AcademicYearID, request.EffectiveLeavingDate.Format("2006-01-02"), *request.FinalizedAt, input.Actor.UserID, input.Actor.Role, request.Status, student.Status})
	if err != nil {
		return FinalizeStudentLeavingRequestOutput{}, err
	}
	if _, err := recordAudit(ctx, repos, RecordAuditInput{Actor: input.Actor, Action: "STUDENT_LEAVING_FINALIZED", EntityType: domain.AuditEntityStudent, EntityID: student.ID, After: payload}); err != nil {
		return FinalizeStudentLeavingRequestOutput{}, err
	}
	if err := uow.Commit(); err != nil {
		return FinalizeStudentLeavingRequestOutput{}, err
	}
	committed = true
	return FinalizeStudentLeavingRequestOutput{Request: &request, Student: &student}, nil
}

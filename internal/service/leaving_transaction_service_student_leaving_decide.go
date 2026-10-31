package service

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type decideForStudentLeavingRequestInput struct {
	Actor                       Actor
	StudentLeavingRequestID     uuid.UUID
	LeavingTransactionRequestID uuid.UUID
	Approve                     bool
	Reason                      string
}

// decideForStudentLeavingRequest memutuskan request tertaut dengan urutan lock lifecycle.
func (s *leavingTransactionRequestService) decideForStudentLeavingRequest(
	ctx context.Context,
	input decideForStudentLeavingRequestInput,
) error {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return err
	}
	if input.StudentLeavingRequestID == uuid.Nil || input.LeavingTransactionRequestID == uuid.Nil {
		return domain.ErrInvalidID
	}

	uow, repos, txCtx, owns, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return fmt.Errorf("begin linked late transaction decision: %w", err)
	}
	ctx = txCtx
	committed := false
	defer func() {
		if owns && !committed {
			_ = uow.Rollback()
		}
	}()
	if repos.AcademicYears == nil || repos.SavingsAccounts == nil || repos.Students == nil ||
		repos.StudentLeavingRequests == nil || repos.LeavingTransactionRequests == nil || repos.AuditLogs == nil {
		return ErrInvalidDependency
	}

	initialRequest, err := repos.LeavingTransactionRequests.GetLeavingTransactionRequest(ctx, input.LeavingTransactionRequestID)
	if err != nil {
		return err
	}
	initialProcess, err := repos.StudentLeavingRequests.GetByID(ctx, input.StudentLeavingRequestID)
	if err != nil {
		return err
	}
	linkedProcessID, err := repos.LeavingTransactionRequests.GetStudentLeavingRequestID(ctx, initialRequest.ID)
	if err != nil {
		return err
	}
	if linkedProcessID == nil || *linkedProcessID != input.StudentLeavingRequestID {
		return domain.ErrInvalidTransition
	}
	initialAccount, err := repos.SavingsAccounts.GetByID(ctx, initialRequest.SavingsAccountID)
	if err != nil {
		return err
	}
	if initialAccount.AcademicYearID != initialRequest.AcademicYearID {
		return domain.ErrInvalidTransition
	}

	for _, yearID := range studentLeavingYearIDs(initialProcess.AcademicYearID, []domain.SavingsAccount{initialAccount}) {
		year, lockErr := repos.AcademicYears.GetByIDForShare(ctx, yearID)
		if lockErr != nil {
			return lockErr
		}
		if year.ID != yearID {
			return repository.ErrConflict
		}
	}
	account, err := repos.SavingsAccounts.GetByIDForUpdate(ctx, initialAccount.ID)
	if err != nil {
		return err
	}
	student, err := repos.Students.GetByIDForUpdate(ctx, initialProcess.StudentID)
	if err != nil {
		return err
	}
	process, err := repos.StudentLeavingRequests.GetByIDForUpdate(ctx, input.StudentLeavingRequestID)
	if err != nil {
		return err
	}
	request, err := repos.LeavingTransactionRequests.GetLeavingTransactionRequestForUpdate(ctx, initialRequest.ID)
	if err != nil {
		return err
	}

	lockedProcessID, err := repos.LeavingTransactionRequests.GetStudentLeavingRequestID(ctx, request.ID)
	if err != nil {
		return err
	}
	if lockedProcessID == nil || *lockedProcessID != process.ID ||
		process.ID != initialProcess.ID || process.StudentID != initialProcess.StudentID ||
		process.AcademicYearID != initialProcess.AcademicYearID ||
		!process.EffectiveLeavingDate.Equal(initialProcess.EffectiveLeavingDate) ||
		request.ID != initialRequest.ID || !reflect.DeepEqual(request, initialRequest) ||
		account.ID != initialAccount.ID || account.StudentID != process.StudentID ||
		account.AcademicYearID != request.AcademicYearID || !account.IsOpen() ||
		!student.IsActive() || student.ID != process.StudentID ||
		process.Status != domain.StudentLeavingInProgress || request.Status != domain.LeavingTransactionPending {
		return domain.ErrInvalidTransition
	}
	year, err := repos.AcademicYears.GetByID(ctx, account.AcademicYearID)
	if err != nil {
		return err
	}
	if year.IsClosed() || dateOnly(request.TransactionDate).Before(dateOnly(year.StartDate)) ||
		!dateOnly(request.TransactionDate).Before(dateOnly(year.EndDate)) ||
		!dateOnly(request.TransactionDate).Before(dateOnly(process.EffectiveLeavingDate)) {
		return domain.ErrInvalidTransition
	}

	before := request
	if err := request.Decide(input.Actor.UserID, input.Approve, input.Reason); err != nil {
		return err
	}
	if err := repos.LeavingTransactionRequests.UpdateLeavingTransactionRequest(ctx, request); err != nil {
		return err
	}
	payload, err := json.Marshal(struct {
		RequestID               uuid.UUID                              `json:"request_id"`
		StudentLeavingRequestID uuid.UUID                              `json:"student_leaving_request_id"`
		StudentID               uuid.UUID                              `json:"student_id"`
		SavingsAccountID        uuid.UUID                              `json:"savings_account_id"`
		AcademicYearID          uuid.UUID                              `json:"academic_year_id"`
		StatusBefore            domain.LeavingTransactionRequestStatus `json:"status_before"`
		StatusAfter             domain.LeavingTransactionRequestStatus `json:"status_after"`
		Approve                 bool                                   `json:"approve"`
		Reason                  string                                 `json:"reason"`
		ActorID                 uuid.UUID                              `json:"actor_id"`
		OccurredAt              string                                 `json:"occurred_at"`
	}{
		RequestID: request.ID, StudentLeavingRequestID: process.ID, StudentID: process.StudentID,
		SavingsAccountID: account.ID, AcademicYearID: account.AcademicYearID,
		StatusBefore: before.Status, StatusAfter: request.Status, Approve: input.Approve,
		Reason: request.DecisionReason, ActorID: input.Actor.UserID, OccurredAt: request.DecidedAt.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return fmt.Errorf("marshal linked late transaction decision audit: %w", err)
	}
	if _, err := recordAudit(ctx, repos, RecordAuditInput{
		Actor: input.Actor, Action: "DECIDE_FOR_STUDENT_LEAVING",
		EntityType: domain.AuditEntityLeavingTransactionRequest, EntityID: request.ID, After: payload,
	}); err != nil {
		return err
	}
	if owns {
		if err := uow.Commit(); err != nil {
			return fmt.Errorf("commit linked late transaction decision: %w", err)
		}
		committed = true
	}
	return nil
}

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

type createLeavingTransactionForStudentLeavingInput struct {
	Actor                     Actor
	StudentLeavingRequestID   uuid.UUID
	SavingsAccountID          uuid.UUID
	Type                      domain.TransactionType
	Amount                    domain.Money
	TransactionDate           time.Time
	Reason, EvidenceReference string
}

type createLeavingTransactionForStudentLeavingOutput struct {
	Request                 domain.LeavingTransactionRequest
	StudentLeavingRequestID uuid.UUID
}

// createForStudentLeavingRequest membuat permintaan tertaut tanpa mengubah ledger atau lifecycle siswa.
func (s *leavingTransactionRequestService) createForStudentLeavingRequest(
	ctx context.Context,
	input createLeavingTransactionForStudentLeavingInput,
) (createLeavingTransactionForStudentLeavingOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleWaliKelas); err != nil {
		return createLeavingTransactionForStudentLeavingOutput{}, err
	}
	if input.StudentLeavingRequestID == uuid.Nil || input.SavingsAccountID == uuid.Nil {
		return createLeavingTransactionForStudentLeavingOutput{}, domain.ErrInvalidID
	}

	uow, repos, txCtx, owns, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return createLeavingTransactionForStudentLeavingOutput{}, fmt.Errorf("begin linked leaving transaction request: %w", err)
	}
	ctx = txCtx
	committed := false
	defer func() {
		if owns && !committed {
			_ = uow.Rollback()
		}
	}()
	if repos.AcademicYears == nil || repos.SavingsAccounts == nil || repos.Students == nil ||
		repos.StudentLeavingRequests == nil || repos.StudentClassHistories == nil ||
		repos.TeacherClassAssignments == nil || repos.LeavingTransactionRequests == nil || repos.AuditLogs == nil {
		return createLeavingTransactionForStudentLeavingOutput{}, ErrInvalidDependency
	}

	// Pra-baca hanya mengumpulkan tahun yang perlu dikunci; semua identitas diperiksa ulang setelah lock.
	initialRequest, err := repos.StudentLeavingRequests.GetByID(ctx, input.StudentLeavingRequestID)
	if err != nil {
		return createLeavingTransactionForStudentLeavingOutput{}, err
	}
	initialAccount, err := repos.SavingsAccounts.GetByID(ctx, input.SavingsAccountID)
	if err != nil {
		return createLeavingTransactionForStudentLeavingOutput{}, err
	}
	if initialRequest.AcademicYearID == uuid.Nil || initialAccount.AcademicYearID == uuid.Nil {
		return createLeavingTransactionForStudentLeavingOutput{}, domain.ErrInvalidID
	}

	yearIDs := studentLeavingYearIDs(initialRequest.AcademicYearID, []domain.SavingsAccount{initialAccount})
	lockedYears := make(map[uuid.UUID]domain.AcademicYear, len(yearIDs))
	for _, yearID := range yearIDs {
		year, lockErr := repos.AcademicYears.GetByIDForShare(ctx, yearID)
		if lockErr != nil {
			return createLeavingTransactionForStudentLeavingOutput{}, lockErr
		}
		if year.ID != yearID {
			return createLeavingTransactionForStudentLeavingOutput{}, repository.ErrConflict
		}
		lockedYears[yearID] = year
	}

	account, err := repos.SavingsAccounts.GetByIDForUpdate(ctx, input.SavingsAccountID)
	if err != nil {
		return createLeavingTransactionForStudentLeavingOutput{}, err
	}
	student, err := repos.Students.GetByIDForUpdate(ctx, account.StudentID)
	if err != nil {
		return createLeavingTransactionForStudentLeavingOutput{}, err
	}
	request, err := repos.StudentLeavingRequests.GetByIDForUpdate(ctx, input.StudentLeavingRequestID)
	if err != nil {
		return createLeavingTransactionForStudentLeavingOutput{}, err
	}

	accountYear := lockedYears[account.AcademicYearID]
	if account.ID != initialAccount.ID || account.StudentID != initialAccount.StudentID ||
		account.AcademicYearID != initialAccount.AcademicYearID ||
		request.ID != initialRequest.ID || request.StudentID != initialRequest.StudentID ||
		request.AcademicYearID != initialRequest.AcademicYearID ||
		!request.EffectiveLeavingDate.Equal(initialRequest.EffectiveLeavingDate) ||
		student.ID != account.StudentID || request.StudentID != student.ID ||
		request.Status != domain.StudentLeavingInProgress || !student.IsActive() ||
		!account.IsOpen() || accountYear.ID != account.AcademicYearID || accountYear.IsClosed() {
		return createLeavingTransactionForStudentLeavingOutput{}, domain.ErrInvalidTransition
	}
	if err := RequireStudentScope(ctx, repos, input.Actor, student.ID, accountYear.ID); err != nil {
		return createLeavingTransactionForStudentLeavingOutput{}, err
	}

	transactionDate := dateOnly(input.TransactionDate)
	if transactionDate.Before(dateOnly(accountYear.StartDate)) ||
		!transactionDate.Before(dateOnly(accountYear.EndDate)) ||
		!transactionDate.Before(dateOnly(request.EffectiveLeavingDate)) {
		return createLeavingTransactionForStudentLeavingOutput{}, domain.ErrInvalidDateRange
	}
	lateRequest, err := domain.NewLeavingTransactionRequest(
		account.ID,
		accountYear.ID,
		input.Type,
		input.Amount,
		transactionDate,
		input.Reason,
		input.EvidenceReference,
		input.Actor.UserID,
	)
	if err != nil {
		return createLeavingTransactionForStudentLeavingOutput{}, err
	}
	usedEvidence, err := repos.LeavingTransactionRequests.HasEvidenceReference(ctx, account.ID, lateRequest.EvidenceReference)
	if err != nil {
		return createLeavingTransactionForStudentLeavingOutput{}, err
	}
	if usedEvidence {
		return createLeavingTransactionForStudentLeavingOutput{}, repository.ErrConflict
	}
	if err := repos.LeavingTransactionRequests.CreateLeavingTransactionRequestForStudentLeaving(ctx, lateRequest, request.ID); err != nil {
		return createLeavingTransactionForStudentLeavingOutput{}, err
	}
	if err := s.auditStudentLeavingRequest(ctx, repos, input.Actor, lateRequest, student.ID, request.ID); err != nil {
		return createLeavingTransactionForStudentLeavingOutput{}, err
	}
	if owns {
		if err := uow.Commit(); err != nil {
			return createLeavingTransactionForStudentLeavingOutput{}, fmt.Errorf("commit linked leaving transaction request: %w", err)
		}
		committed = true
	}
	return createLeavingTransactionForStudentLeavingOutput{Request: lateRequest, StudentLeavingRequestID: request.ID}, nil
}

func (s *leavingTransactionRequestService) auditStudentLeavingRequest(
	ctx context.Context,
	repos repository.RepositorySet,
	actor Actor,
	request domain.LeavingTransactionRequest,
	studentID uuid.UUID,
	studentLeavingRequestID uuid.UUID,
) error {
	payload, err := json.Marshal(struct {
		RequestID               uuid.UUID                              `json:"request_id"`
		StudentLeavingRequestID uuid.UUID                              `json:"student_leaving_request_id"`
		StudentID               uuid.UUID                              `json:"student_id"`
		SavingsAccountID        uuid.UUID                              `json:"savings_account_id"`
		AcademicYearID          uuid.UUID                              `json:"academic_year_id"`
		Type                    domain.TransactionType                 `json:"type"`
		Amount                  domain.Money                           `json:"amount"`
		TransactionDate         string                                 `json:"transaction_date"`
		Reason                  string                                 `json:"reason"`
		EvidenceReference       string                                 `json:"evidence_reference"`
		ActorID                 uuid.UUID                              `json:"actor_id"`
		ActorRole               domain.UserRole                        `json:"actor_role"`
		Status                  domain.LeavingTransactionRequestStatus `json:"status"`
		OccurredAt              time.Time                              `json:"occurred_at"`
	}{
		RequestID:               request.ID,
		StudentLeavingRequestID: studentLeavingRequestID,
		StudentID:               studentID,
		SavingsAccountID:        request.SavingsAccountID,
		AcademicYearID:          request.AcademicYearID,
		Type:                    request.Type,
		Amount:                  request.Amount,
		TransactionDate:         request.TransactionDate.Format("2006-01-02"),
		Reason:                  request.Reason,
		EvidenceReference:       request.EvidenceReference,
		ActorID:                 actor.UserID,
		ActorRole:               actor.Role,
		Status:                  request.Status,
		OccurredAt:              request.RequestedAt,
	})
	if err != nil {
		return fmt.Errorf("marshal linked leaving transaction request audit: %w", err)
	}
	_, err = recordAudit(ctx, repos, RecordAuditInput{
		Actor:      actor,
		Action:     "REQUEST_FOR_STUDENT_LEAVING",
		EntityType: domain.AuditEntityLeavingTransactionRequest,
		EntityID:   request.ID,
		After:      payload,
	})
	return err
}

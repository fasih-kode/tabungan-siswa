package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

func (s *studentLeavingRequestService) Cancel(
	ctx context.Context,
	input CancelStudentLeavingRequestInput,
) (CancelStudentLeavingRequestOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return CancelStudentLeavingRequestOutput{}, err
	}
	if input.StudentLeavingRequestID == uuid.Nil {
		return CancelStudentLeavingRequestOutput{}, domain.ErrInvalidID
	}
	if s.deps.Repositories.StudentLeavingRequests == nil || s.deps.Repositories.SavingsAccounts == nil {
		return CancelStudentLeavingRequestOutput{}, ErrInvalidDependency
	}

	initialRequest, err := s.deps.Repositories.StudentLeavingRequests.GetByID(ctx, input.StudentLeavingRequestID)
	if err != nil {
		return CancelStudentLeavingRequestOutput{}, err
	}
	initialAccounts, err := s.deps.Repositories.SavingsAccounts.ListByStudent(ctx, initialRequest.StudentID)
	if err != nil {
		return CancelStudentLeavingRequestOutput{}, err
	}
	initialAccounts = orderedStudentLeavingAccounts(initialAccounts)
	yearIDs := studentLeavingYearIDs(initialRequest.AcademicYearID, initialAccounts)

	uow, err := s.deps.UOW.Begin(ctx)
	if err != nil {
		return CancelStudentLeavingRequestOutput{}, fmt.Errorf("begin student leaving cancellation: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()
	repos := uow.Repositories()
	if repos.AcademicYears == nil || repos.SavingsAccounts == nil || repos.Students == nil ||
		repos.StudentLeavingRequests == nil || repos.Settlements == nil || repos.AuditLogs == nil {
		return CancelStudentLeavingRequestOutput{}, ErrInvalidDependency
	}

	for _, yearID := range yearIDs {
		year, lockErr := repos.AcademicYears.GetByIDForShare(ctx, yearID)
		if lockErr != nil {
			return CancelStudentLeavingRequestOutput{}, lockErr
		}
		if year.ID != yearID {
			return CancelStudentLeavingRequestOutput{}, repository.ErrConflict
		}
	}

	lockedAccounts, err := repos.SavingsAccounts.ListByStudentForUpdate(ctx, initialRequest.StudentID)
	if err != nil {
		return CancelStudentLeavingRequestOutput{}, err
	}
	lockedAccounts = orderedStudentLeavingAccounts(lockedAccounts)
	if !sameStudentLeavingAccounts(initialAccounts, lockedAccounts) {
		return CancelStudentLeavingRequestOutput{}, repository.ErrConflict
	}

	student, err := repos.Students.GetByIDForUpdate(ctx, initialRequest.StudentID)
	if err != nil {
		return CancelStudentLeavingRequestOutput{}, err
	}
	if student.ID != initialRequest.StudentID || !student.IsActive() {
		return CancelStudentLeavingRequestOutput{}, domain.ErrInvalidTransition
	}

	request, err := repos.StudentLeavingRequests.GetByIDForUpdate(ctx, input.StudentLeavingRequestID)
	if err != nil {
		return CancelStudentLeavingRequestOutput{}, err
	}
	if request.ID != initialRequest.ID || request.StudentID != initialRequest.StudentID ||
		request.AcademicYearID != initialRequest.AcademicYearID ||
		!request.EffectiveLeavingDate.Equal(initialRequest.EffectiveLeavingDate) ||
		request.Status != domain.StudentLeavingInProgress {
		return CancelStudentLeavingRequestOutput{}, domain.ErrInvalidTransition
	}

	currentAccounts, err := repos.SavingsAccounts.ListByStudent(ctx, student.ID)
	if err != nil {
		return CancelStudentLeavingRequestOutput{}, err
	}
	if !sameStudentLeavingAccounts(lockedAccounts, orderedStudentLeavingAccounts(currentAccounts)) {
		return CancelStudentLeavingRequestOutput{}, repository.ErrConflict
	}

	if err := validateStudentLeavingCancellationFinance(ctx, repos, request, lockedAccounts); err != nil {
		return CancelStudentLeavingRequestOutput{}, err
	}

	cancelledAt := s.now().UTC()
	if err := request.Cancel(input.Actor.UserID, cancelledAt, input.Reason); err != nil {
		return CancelStudentLeavingRequestOutput{}, err
	}
	if err := repos.StudentLeavingRequests.Update(ctx, request); err != nil {
		return CancelStudentLeavingRequestOutput{}, err
	}

	payload, err := json.Marshal(struct {
		StudentLeavingRequestID uuid.UUID                          `json:"student_leaving_request_id"`
		StudentID               uuid.UUID                          `json:"student_id"`
		AcademicYearID          uuid.UUID                          `json:"academic_year_id"`
		Reason                  string                             `json:"reason"`
		Status                  domain.StudentLeavingRequestStatus `json:"status"`
		ActorID                 uuid.UUID                          `json:"actor_id"`
		ActorRole               domain.UserRole                    `json:"actor_role"`
		OccurredAt              time.Time                          `json:"occurred_at"`
	}{
		StudentLeavingRequestID: request.ID,
		StudentID:               request.StudentID,
		AcademicYearID:          request.AcademicYearID,
		Reason:                  *request.CancellationReason,
		Status:                  request.Status,
		ActorID:                 input.Actor.UserID,
		ActorRole:               input.Actor.Role,
		OccurredAt:              cancelledAt,
	})
	if err != nil {
		return CancelStudentLeavingRequestOutput{}, err
	}
	if _, err := recordAudit(ctx, repos, RecordAuditInput{
		Actor:      input.Actor,
		Action:     "STUDENT_LEAVING_CANCELLED",
		EntityType: domain.AuditEntityStudent,
		EntityID:   student.ID,
		After:      payload,
	}); err != nil {
		return CancelStudentLeavingRequestOutput{}, err
	}
	if err := uow.Commit(); err != nil {
		return CancelStudentLeavingRequestOutput{}, fmt.Errorf("commit student leaving cancellation: %w", err)
	}
	committed = true
	return CancelStudentLeavingRequestOutput{Request: &request}, nil
}

func orderedStudentLeavingAccounts(accounts []domain.SavingsAccount) []domain.SavingsAccount {
	ordered := append([]domain.SavingsAccount(nil), accounts...)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].ID.String() < ordered[j].ID.String()
	})
	return ordered
}

func studentLeavingYearIDs(referenceYearID uuid.UUID, accounts []domain.SavingsAccount) []uuid.UUID {
	yearSet := map[uuid.UUID]struct{}{referenceYearID: {}}
	for _, account := range accounts {
		yearSet[account.AcademicYearID] = struct{}{}
	}
	yearIDs := make([]uuid.UUID, 0, len(yearSet))
	for yearID := range yearSet {
		yearIDs = append(yearIDs, yearID)
	}
	sort.Slice(yearIDs, func(i, j int) bool { return yearIDs[i].String() < yearIDs[j].String() })
	return yearIDs
}

func sameStudentLeavingAccounts(left, right []domain.SavingsAccount) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].ID != right[i].ID || left[i].StudentID != right[i].StudentID ||
			left[i].AcademicYearID != right[i].AcademicYearID {
			return false
		}
	}
	return true
}

func validateStudentLeavingCancellationFinance(
	ctx context.Context,
	repos repository.RepositorySet,
	request domain.StudentLeavingRequest,
	accounts []domain.SavingsAccount,
) error {
	issues, err := repos.StudentLeavingRequests.ListFinancialLinkIssues(ctx, request.ID)
	if err != nil {
		return err
	}
	if len(issues) != 0 {
		return domain.ErrInvalidTransition
	}

	accountsByID := make(map[uuid.UUID]domain.SavingsAccount, len(accounts))
	for _, account := range accounts {
		if account.StudentID != request.StudentID || account.ID == uuid.Nil || account.AcademicYearID == uuid.Nil {
			return domain.ErrInvalidTransition
		}
		accountsByID[account.ID] = account
	}

	settlements, err := repos.Settlements.ListByStudentLeavingRequest(ctx, request.ID)
	if err != nil {
		return err
	}
	for _, settlement := range settlements {
		account, ok := accountsByID[settlement.SavingsAccountID]
		if settlement.ID == uuid.Nil || settlement.ExecutedBy == uuid.Nil || settlement.ExecutedAt.IsZero() ||
			!ok || settlement.Type != domain.SettlementStudentLeaving ||
			settlement.Status != domain.SettlementCompleted || !account.IsSettled() ||
			settlement.Amount < 0 || settlement.Amount > domain.MaxMoney {
			return domain.ErrInvalidTransition
		}

		movements, err := repos.Settlements.ListCashMovements(ctx, settlement.ID)
		if err != nil {
			return err
		}
		if err := validateStudentLeavingHandover(settlement.Amount, movements); err != nil {
			return err
		}
	}
	return nil
}

func validateStudentLeavingHandover(
	settlementAmount domain.Money,
	movements []domain.SettlementCashMovement,
) error {
	var net domain.Money
	for _, movement := range movements {
		if movement.ID == uuid.Nil || movement.SavingsSettlementID == uuid.Nil ||
			movement.ConfirmedBy == uuid.Nil || movement.ConfirmedAt.IsZero() ||
			movement.Amount <= 0 || movement.Amount > domain.MaxMoney {
			return domain.ErrInvalidTransition
		}
		switch movement.Kind {
		case domain.SettlementCashHandover:
			if int64(net) > math.MaxInt64-int64(movement.Amount) {
				return domain.ErrInvalidTransition
			}
			net += movement.Amount
		case domain.SettlementCashReturn:
			if int64(net) < math.MinInt64+int64(movement.Amount) {
				return domain.ErrInvalidTransition
			}
			net -= movement.Amount
		default:
			return domain.ErrInvalidTransition
		}
	}
	if net < 0 || net != settlementAmount {
		return domain.ErrInvalidTransition
	}
	return nil
}

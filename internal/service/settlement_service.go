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

var _ SettlementService = (*settlementService)(nil)

func (s *settlementService) createReopenRequestAudit(ctx context.Context, repos repository.RepositorySet, actor Actor, action string, request domain.SettlementReopenRequest, before *domain.SettlementReopenRequest) error {
	var b []byte
	var err error
	if before != nil {
		b, err = json.Marshal(before)
		if err != nil {
			return err
		}
	}
	a, err := json.Marshal(request)
	if err != nil {
		return err
	}
	_, err = recordAudit(ctx, repos, RecordAuditInput{Actor: actor, Action: action, EntityType: domain.AuditEntitySettlementReopenRequest, EntityID: request.ID, Before: b, After: a})
	return err
}
func (s *settlementService) createAcademicYearAudit(ctx context.Context, repos repository.RepositorySet, actor Actor, action string, year domain.AcademicYear, before *domain.AcademicYear) error {
	var b []byte
	var err error
	if before != nil {
		b, err = json.Marshal(before)
		if err != nil {
			return err
		}
	}
	a, err := json.Marshal(year)
	if err != nil {
		return err
	}
	_, err = recordAudit(ctx, repos, RecordAuditInput{Actor: actor, Action: action, EntityType: domain.AuditEntityAcademicYear, EntityID: year.ID, Before: b, After: a})
	return err
}

func (s *settlementService) authorizeAccount(
	ctx context.Context,
	repos repository.RepositorySet,
	actor Actor,
	account domain.SavingsAccount,
) error {
	switch actor.Role {
	case domain.RoleAdmin:
		return actor.Validate()

	case domain.RoleWaliKelas:
		return RequireStudentScope(
			ctx,
			repos,
			actor,
			account.StudentID,
			account.AcademicYearID,
		)

	case domain.RoleSiswa:
		student, err := repos.Students.GetByID(ctx, account.StudentID)
		if err != nil {
			return err
		}
		if student.UserID == nil {
			return ErrScopeViolation
		}
		return RequireSelf(actor, *student.UserID)

	default:
		return ErrForbidden
	}
}

func (s *settlementService) createSettlementAudit(
	ctx context.Context,
	repos repository.RepositorySet,
	actor Actor,
	action string,
	settlement domain.SavingsSettlement,
	before *domain.SavingsSettlement,
) error {
	var beforeData []byte
	var err error
	if before != nil {
		beforeData, err = json.Marshal(before)
		if err != nil {
			return fmt.Errorf("marshal settlement audit before data: %w", err)
		}
	}

	afterData, err := json.Marshal(settlement)
	if err != nil {
		return fmt.Errorf("marshal settlement audit after data: %w", err)
	}

	_, err = recordAudit(ctx, repos, RecordAuditInput{
		Actor:      actor,
		Action:     action,
		EntityType: domain.AuditEntitySavingsSettlement,
		EntityID:   settlement.ID,
		Before:     beforeData,
		After:      afterData,
	})
	return err
}

func (s *settlementService) createSavingsAccountAudit(
	ctx context.Context,
	repos repository.RepositorySet,
	actor Actor,
	action string,
	account domain.SavingsAccount,
	before *domain.SavingsAccount,
) error {
	var beforeData []byte
	var err error
	if before != nil {
		beforeData, err = json.Marshal(before)
		if err != nil {
			return fmt.Errorf("marshal savings account audit before data: %w", err)
		}
	}

	afterData, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("marshal savings account audit after data: %w", err)
	}

	_, err = recordAudit(ctx, repos, RecordAuditInput{
		Actor:      actor,
		Action:     action,
		EntityType: domain.AuditEntitySavingsAccount,
		EntityID:   account.ID,
		Before:     beforeData,
		After:      afterData,
	})
	return err
}

func (s *settlementService) ConfirmHandover(ctx context.Context, input ConfirmSettlementHandoverInput) (ConfirmSettlementHandoverOutput, error) {
	if !input.Confirm {
		return ConfirmSettlementHandoverOutput{}, domain.ErrInvalidValue
	}
	if err := RequireRole(input.Actor, domain.RoleWaliKelas); err != nil {
		return ConfirmSettlementHandoverOutput{}, err
	}
	if input.SettlementID == uuid.Nil {
		return ConfirmSettlementHandoverOutput{}, domain.ErrInvalidID
	}
	uow, repos, ctx, owns, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return ConfirmSettlementHandoverOutput{}, err
	}
	committed := false
	defer func() {
		if owns && !committed {
			_ = uow.Rollback()
		}
	}()
	settlement, err := repos.Settlements.GetByID(ctx, input.SettlementID)
	if err != nil {
		return ConfirmSettlementHandoverOutput{}, err
	}
	unlocked, err := repos.SavingsAccounts.GetByID(ctx, settlement.SavingsAccountID)
	if err != nil {
		return ConfirmSettlementHandoverOutput{}, err
	}
	year, err := repos.AcademicYears.GetByIDForShare(ctx, unlocked.AcademicYearID)
	if err != nil {
		return ConfirmSettlementHandoverOutput{}, err
	}
	account, err := repos.SavingsAccounts.GetByIDForUpdate(ctx, unlocked.ID)
	if err != nil {
		return ConfirmSettlementHandoverOutput{}, err
	}
	if year.IsClosed() || (!year.IsClosing() && settlement.Type != domain.SettlementStudentLeaving) || !account.IsSettled() || !settlement.IsCompleted() {
		return ConfirmSettlementHandoverOutput{}, domain.ErrInvalidTransition
	}
	if err := authorizeResponsibleWali(ctx, repos, input.Actor, account, year); err != nil {
		return ConfirmSettlementHandoverOutput{}, err
	}
	if settlement.Amount <= 0 {
		return ConfirmSettlementHandoverOutput{}, domain.ErrInvalidTransition
	}
	movements, err := repos.Settlements.ListCashMovements(ctx, settlement.ID)
	if err != nil {
		return ConfirmSettlementHandoverOutput{}, err
	}
	var delivered domain.Money
	for _, m := range movements {
		if m.Kind == domain.SettlementCashHandover {
			delivered += m.Amount
		} else {
			delivered -= m.Amount
		}
	}
	delta := settlement.Amount - delivered
	if delta == 0 {
		return ConfirmSettlementHandoverOutput{}, repository.ErrConflict
	}
	kind, amount := domain.SettlementCashHandover, delta
	if delta < 0 {
		kind, amount = domain.SettlementCashReturn, -delta
	}
	movement, err := domain.NewSettlementCashMovement(settlement.ID, kind, amount, input.Actor.UserID, time.Now().UTC(), input.ReceiptReference)
	if err != nil {
		return ConfirmSettlementHandoverOutput{}, err
	}
	if err := repos.Settlements.CreateCashMovement(ctx, movement); err != nil {
		return ConfirmSettlementHandoverOutput{}, err
	}
	student, err := repos.Students.GetByID(ctx, account.StudentID)
	if err != nil {
		return ConfirmSettlementHandoverOutput{}, err
	}
	if err := s.createSettlementMovementAudit(ctx, repos, input.Actor, settlement, movement, student.ID); err != nil {
		return ConfirmSettlementHandoverOutput{}, err
	}
	if owns {
		if err := uow.Commit(); err != nil {
			return ConfirmSettlementHandoverOutput{}, err
		}
		committed = true
	}
	return ConfirmSettlementHandoverOutput{Movement: &movement}, nil
}

func hasPendingLateRequest(ctx context.Context, repos repository.RepositorySet, accountID uuid.UUID) (bool, error) {
	if repos.LeavingTransactionRequests == nil {
		return false, nil
	}
	return repos.LeavingTransactionRequests.HasPendingForSavingsAccount(ctx, accountID)
}

func authorizeResponsibleWali(ctx context.Context, repos repository.RepositorySet, actor Actor, account domain.SavingsAccount, year domain.AcademicYear) error {
	if actor.Role == domain.RoleAdmin {
		return actor.Validate()
	}
	if actor.Role != domain.RoleWaliKelas {
		return ErrForbidden
	}
	finalDate := year.EndDate.AddDate(0, 0, -1)
	history, err := repos.StudentClassHistories.GetCurrentByStudentAndAcademicYear(ctx, account.StudentID, year.ID, finalDate)
	if err != nil {
		return ErrScopeViolation
	}
	assignments, err := repos.TeacherClassAssignments.ListByClassAndAcademicYear(ctx, history.ClassID, year.ID, repository.ListOptions{Limit: 100})
	if err != nil {
		return err
	}
	for _, a := range assignments {
		if a.UserID == actor.UserID && a.Contains(finalDate) {
			return nil
		}
	}
	return ErrScopeViolation
}

func correctionContextForApprovedRequest(ctx context.Context, repos repository.RepositorySet, actor Actor, year domain.AcademicYear, accountID uuid.UUID) (context.Context, error) {
	if year.IsOpen() {
		return ctx, nil
	}
	if !year.IsClosing() {
		return nil, domain.ErrInvalidTransition
	}
	if scope, ok := approvedCorrectionScopeFromContext(ctx); ok && scope.AccountID == accountID {
		return ctx, nil
	}
	if repos.Settlements == nil {
		return nil, domain.ErrInvalidTransition
	}
	request, err := repos.Settlements.GetApprovedReopenRequestByAccount(ctx, accountID)
	if err != nil {
		return nil, domain.ErrInvalidTransition
	}
	if request.Status != domain.SettlementReopenApproved || request.SavingsAccountID != accountID || request.AcademicYearID != year.ID {
		return nil, domain.ErrInvalidTransition
	}
	account, err := repos.SavingsAccounts.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if err := authorizeResponsibleWali(ctx, repos, actor, account, year); err != nil {
		return nil, err
	}
	return withApprovedCorrectionScope(ctx, request.ID, accountID), nil
}

func (s *settlementService) createSettlementMovementAudit(ctx context.Context, repos repository.RepositorySet, actor Actor, settlement domain.SavingsSettlement, movement domain.SettlementCashMovement, studentID uuid.UUID) error {
	after, err := json.Marshal(struct {
		Settlement         domain.SavingsSettlement      `json:"settlement"`
		Movement           domain.SettlementCashMovement `json:"movement"`
		RecipientStudentID uuid.UUID                     `json:"recipient_student_id"`
	}{settlement, movement, studentID})
	if err != nil {
		return err
	}
	_, err = recordAudit(ctx, repos, RecordAuditInput{Actor: actor, Action: string(movement.Kind), EntityType: domain.AuditEntitySavingsSettlement, EntityID: settlement.ID, After: after})
	return err
}

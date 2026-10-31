package service

import (
	"context"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

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

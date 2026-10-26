package service

import (
	"context"
	"errors"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

func (s *settlementService) RequestReopen(ctx context.Context, input RequestSettlementReopenInput) (RequestSettlementReopenOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleWaliKelas); err != nil {
		return RequestSettlementReopenOutput{}, err
	}
	if input.SettlementID == uuid.Nil {
		return RequestSettlementReopenOutput{}, domain.ErrInvalidID
	}
	reason, err := validatedCorrectionReason(input.Reason)
	if err != nil {
		return RequestSettlementReopenOutput{}, err
	}
	uow, repos, ctx, owns, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return RequestSettlementReopenOutput{}, err
	}
	committed := false
	defer func() {
		if owns && !committed {
			_ = uow.Rollback()
		}
	}()
	settlement, err := repos.Settlements.GetByID(ctx, input.SettlementID)
	if err != nil {
		return RequestSettlementReopenOutput{}, err
	}
	unlocked, err := repos.SavingsAccounts.GetByID(ctx, settlement.SavingsAccountID)
	if err != nil {
		return RequestSettlementReopenOutput{}, err
	}
	year, err := repos.AcademicYears.GetByIDForUpdate(ctx, unlocked.AcademicYearID)
	if err != nil {
		return RequestSettlementReopenOutput{}, err
	}
	if year.IsClosed() == false || settlement.Type != domain.SettlementRegularYearEnd || !settlement.IsCompleted() {
		return RequestSettlementReopenOutput{}, domain.ErrInvalidTransition
	}
	successor, err := repos.AcademicYears.HasSuccessor(ctx, year.ID)
	if err != nil {
		return RequestSettlementReopenOutput{}, err
	}
	if successor {
		return RequestSettlementReopenOutput{}, domain.ErrInvalidTransition
	}
	account, err := repos.SavingsAccounts.GetByIDForUpdate(ctx, unlocked.ID)
	if err != nil {
		return RequestSettlementReopenOutput{}, err
	}
	if !account.IsSettled() {
		return RequestSettlementReopenOutput{}, domain.ErrInvalidTransition
	}
	if err := authorizeResponsibleWali(ctx, repos, input.Actor, account, year); err != nil {
		return RequestSettlementReopenOutput{}, err
	}
	if _, err := repos.Settlements.GetCompletedBySavingsAccount(ctx, account.ID); err != nil {
		return RequestSettlementReopenOutput{}, err
	}
	if active, err := repos.Settlements.GetActiveReopenRequest(ctx, settlement.ID); err == nil && active.ID != uuid.Nil {
		return RequestSettlementReopenOutput{}, repository.ErrConflict
	} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return RequestSettlementReopenOutput{}, err
	}
	now := time.Now().UTC()
	request := domain.SettlementReopenRequest{ID: uuid.New(), SavingsSettlementID: settlement.ID, SavingsAccountID: account.ID, AcademicYearID: year.ID, RequestedBy: input.Actor.UserID, Reason: reason, RequestedAt: now, Status: domain.SettlementReopenPending}
	if err := repos.Settlements.CreateReopenRequest(ctx, request); err != nil {
		return RequestSettlementReopenOutput{}, err
	}
	if err := s.createReopenRequestAudit(ctx, repos, input.Actor, "REQUEST", request, nil); err != nil {
		return RequestSettlementReopenOutput{}, err
	}
	if owns {
		if err := uow.Commit(); err != nil {
			return RequestSettlementReopenOutput{}, err
		}
		committed = true
	}
	return RequestSettlementReopenOutput{Request: &request}, nil
}

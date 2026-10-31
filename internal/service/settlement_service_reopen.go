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

func (s *settlementService) DecideReopen(ctx context.Context, input DecideSettlementReopenInput) (DecideSettlementReopenOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return DecideSettlementReopenOutput{}, err
	}
	if input.RequestID == uuid.Nil {
		return DecideSettlementReopenOutput{}, domain.ErrInvalidID
	}
	reason, err := validatedCorrectionReason(input.Reason)
	if err != nil {
		return DecideSettlementReopenOutput{}, err
	}
	uow, repos, ctx, owns, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return DecideSettlementReopenOutput{}, err
	}
	committed := false
	defer func() {
		if owns && !committed {
			_ = uow.Rollback()
		}
	}()
	unlocked, err := repos.Settlements.GetReopenRequest(ctx, input.RequestID)
	if err != nil {
		return DecideSettlementReopenOutput{}, err
	}
	year, err := repos.AcademicYears.GetByIDForUpdate(ctx, unlocked.AcademicYearID)
	if err != nil {
		return DecideSettlementReopenOutput{}, err
	}
	successor, err := repos.AcademicYears.HasSuccessor(ctx, year.ID)
	if err != nil {
		return DecideSettlementReopenOutput{}, err
	}
	if successor {
		return DecideSettlementReopenOutput{}, domain.ErrInvalidTransition
	}
	request, err := repos.Settlements.GetReopenRequestForUpdate(ctx, input.RequestID)
	if err != nil {
		return DecideSettlementReopenOutput{}, err
	}
	if request.Status != domain.SettlementReopenPending || request.RequestedBy == input.Actor.UserID {
		return DecideSettlementReopenOutput{}, repository.ErrConflict
	}
	if !year.IsClosed() {
		return DecideSettlementReopenOutput{}, domain.ErrInvalidTransition
	}
	now := time.Now().UTC()
	actorID := input.Actor.UserID
	request.DecidedBy = &actorID
	request.DecidedAt = &now
	request.DecisionReason = &reason
	if input.Approve {
		request.Status = domain.SettlementReopenApproved
	} else {
		request.Status = domain.SettlementReopenRejected
	}
	if err := repos.Settlements.UpdateReopenRequest(ctx, request); err != nil {
		return DecideSettlementReopenOutput{}, err
	}
	decision := "REJECT"
	if input.Approve {
		decision = "APPROVE"
	}
	if err := s.createReopenRequestAudit(ctx, repos, input.Actor, decision, request, nil); err != nil {
		return DecideSettlementReopenOutput{}, err
	}
	if owns {
		if err := uow.Commit(); err != nil {
			return DecideSettlementReopenOutput{}, err
		}
		committed = true
	}
	return DecideSettlementReopenOutput{Request: &request}, nil
}

func (s *settlementService) Reopen(ctx context.Context, input ReopenSettlementInput) (ReopenSettlementOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleWaliKelas); err != nil {
		return ReopenSettlementOutput{}, err
	}
	if input.SettlementID == uuid.Nil {
		return ReopenSettlementOutput{}, domain.ErrInvalidID
	}
	uow, repos, ctx, owns, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return ReopenSettlementOutput{}, err
	}
	committed := false
	defer func() {
		if owns && !committed {
			_ = uow.Rollback()
		}
	}()
	settlement, err := repos.Settlements.GetByID(ctx, input.SettlementID)
	if err != nil {
		return ReopenSettlementOutput{}, err
	}
	unlocked, err := repos.SavingsAccounts.GetByID(ctx, settlement.SavingsAccountID)
	if err != nil {
		return ReopenSettlementOutput{}, err
	}
	year, err := repos.AcademicYears.GetByIDForUpdate(ctx, unlocked.AcademicYearID)
	if err != nil {
		return ReopenSettlementOutput{}, err
	}
	successor, err := repos.AcademicYears.HasSuccessor(ctx, year.ID)
	if err != nil {
		return ReopenSettlementOutput{}, err
	}
	if successor {
		return ReopenSettlementOutput{}, domain.ErrInvalidTransition
	}
	account, err := repos.SavingsAccounts.GetByIDForUpdate(ctx, unlocked.ID)
	if err != nil {
		return ReopenSettlementOutput{}, err
	}
	request, err := repos.Settlements.GetActiveReopenRequest(ctx, settlement.ID)
	if err != nil {
		return ReopenSettlementOutput{}, err
	}
	request, err = repos.Settlements.GetReopenRequestForUpdate(ctx, request.ID)
	if err != nil {
		return ReopenSettlementOutput{}, err
	}
	if request.Status != domain.SettlementReopenApproved || !year.IsClosed() || !account.IsSettled() || !settlement.IsCompleted() {
		return ReopenSettlementOutput{}, domain.ErrInvalidTransition
	}
	if err := authorizeResponsibleWali(ctx, repos, input.Actor, account, year); err != nil {
		return ReopenSettlementOutput{}, err
	}
	current, err := repos.Settlements.GetCompletedBySavingsAccount(ctx, account.ID)
	if err != nil {
		return ReopenSettlementOutput{}, err
	}
	if current.ID != settlement.ID {
		return ReopenSettlementOutput{}, repository.ErrConflict
	}
	beforeYear := year
	beforeSettlement := settlement
	beforeAccount := account
	if err := year.ReopenForCorrection(); err != nil {
		return ReopenSettlementOutput{}, err
	}
	if err := settlement.Supersede(); err != nil {
		return ReopenSettlementOutput{}, err
	}
	if err := account.Reopen(); err != nil {
		return ReopenSettlementOutput{}, err
	}
	if err := repos.AcademicYears.Update(ctx, year); err != nil {
		return ReopenSettlementOutput{}, err
	}
	if err := repos.Settlements.Update(ctx, settlement); err != nil {
		return ReopenSettlementOutput{}, err
	}
	if err := repos.SavingsAccounts.Update(ctx, account); err != nil {
		return ReopenSettlementOutput{}, err
	}
	if err := s.createSettlementAudit(ctx, repos, input.Actor, "REOPEN", settlement, &beforeSettlement); err != nil {
		return ReopenSettlementOutput{}, err
	}
	if err := s.createSavingsAccountAudit(ctx, repos, input.Actor, "REOPEN", account, &beforeAccount); err != nil {
		return ReopenSettlementOutput{}, err
	}
	if err := s.createAcademicYearAudit(ctx, repos, input.Actor, "REOPEN_FOR_CORRECTION", year, &beforeYear); err != nil {
		return ReopenSettlementOutput{}, err
	}
	if err := s.createReopenRequestAudit(ctx, repos, input.Actor, "EXECUTE", request, nil); err != nil {
		return ReopenSettlementOutput{}, err
	}
	if owns {
		if err := uow.Commit(); err != nil {
			return ReopenSettlementOutput{}, err
		}
		committed = true
	}
	return ReopenSettlementOutput{PreviousSettlement: &settlement, Account: &account}, nil
}

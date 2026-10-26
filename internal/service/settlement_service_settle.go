package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

func (s *settlementService) SettleYearEnd(
	ctx context.Context,
	input SettleYearEndInput,
) (SettleYearEndOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleWaliKelas); err != nil {
		return SettleYearEndOutput{}, err
	}

	if input.SavingsAccountID == uuid.Nil {
		return SettleYearEndOutput{}, domain.ErrInvalidID
	}

	uow, repos, txCtx, ownsTransaction, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return SettleYearEndOutput{}, fmt.Errorf("begin settle year end transaction: %w", err)
	}

	ctx = txCtx
	committed := false
	defer func() {
		if ownsTransaction && !committed {
			_ = uow.Rollback()
		}
	}()

	// FASE 21 lock order: immutable year ID discovery, academic-year FOR SHARE,
	// then the account FOR UPDATE. This serializes settlement with close and
	// with transaction writes without an account-then-year cycle.
	unlockedAccount, err := repos.SavingsAccounts.GetByID(ctx, input.SavingsAccountID)
	if err != nil {
		return SettleYearEndOutput{}, err
	}
	academicYear, err := repos.AcademicYears.GetByIDForShare(ctx, unlockedAccount.AcademicYearID)
	if err != nil {
		return SettleYearEndOutput{}, err
	}
	account, err := repos.SavingsAccounts.GetByIDForUpdate(ctx, input.SavingsAccountID)
	if err != nil {
		return SettleYearEndOutput{}, err
	}
	if account.AcademicYearID != academicYear.ID {
		return SettleYearEndOutput{}, repository.ErrConflict
	}

	if err := authorizeResponsibleWali(ctx, repos, input.Actor, account, academicYear); err != nil {
		return SettleYearEndOutput{}, err
	}

	if !account.IsOpen() {
		return SettleYearEndOutput{}, domain.ErrInvalidTransition
	}

	if !academicYear.IsClosing() {
		return SettleYearEndOutput{}, domain.ErrInvalidTransition
	}
	if pending, e := hasPendingLateRequest(ctx, repos, account.ID); e != nil {
		return SettleYearEndOutput{}, e
	} else if pending {
		return SettleYearEndOutput{}, domain.ErrInvalidTransition
	}

	transactions, err := repos.Transactions.ListActiveBySavingsAccount(ctx, account.ID)
	if err != nil {
		return SettleYearEndOutput{}, err
	}

	balance, err := domain.CalculateBalance(transactions)
	if err != nil {
		return SettleYearEndOutput{}, err
	}
	if balance.Amount != input.ExpectedBalance {
		return SettleYearEndOutput{}, repository.ErrConflict
	}

	var replacesID *uuid.UUID
	var reopenRequest *domain.SettlementReopenRequest
	request, requestErr := repos.Settlements.GetApprovedReopenRequestByAccount(ctx, account.ID)
	if requestErr == nil {
		if request.AcademicYearID != academicYear.ID || request.Status != domain.SettlementReopenApproved {
			return SettleYearEndOutput{}, domain.ErrInvalidTransition
		}
		replacesID = &request.SavingsSettlementID
		reopenRequest = &request
	} else if !errors.Is(requestErr, repository.ErrNotFound) {
		return SettleYearEndOutput{}, requestErr
	}
	now := time.Now()
	settlement, err := domain.NewSavingsSettlement(
		account.ID,
		domain.SettlementRegularYearEnd,
		balance.Amount,
		input.Actor.UserID,
		now,
		replacesID,
	)
	if err != nil {
		return SettleYearEndOutput{}, err
	}

	beforeAccount := account
	if err := account.Settle(settlement.ExecutedAt); err != nil {
		return SettleYearEndOutput{}, err
	}

	if err := repos.SavingsAccounts.Update(ctx, account); err != nil {
		return SettleYearEndOutput{}, err
	}

	if err := repos.Settlements.Create(ctx, settlement); err != nil {
		return SettleYearEndOutput{}, err
	}
	if reopenRequest != nil {
		reopenRequest.Status = domain.SettlementReopenComplete
		reopenRequest.CompletedAt = &now
		if err := repos.Settlements.UpdateReopenRequest(ctx, *reopenRequest); err != nil {
			return SettleYearEndOutput{}, err
		}
		if err := s.createReopenRequestAudit(ctx, repos, input.Actor, "COMPLETE", *reopenRequest, nil); err != nil {
			return SettleYearEndOutput{}, err
		}
	}

	if err := s.createSettlementAudit(ctx, repos, input.Actor, "SETTLE", settlement, nil); err != nil {
		return SettleYearEndOutput{}, err
	}

	if err := s.createSavingsAccountAudit(ctx, repos, input.Actor, "SETTLE", account, &beforeAccount); err != nil {
		return SettleYearEndOutput{}, err
	}

	if ownsTransaction {
		if err := uow.Commit(); err != nil {
			return SettleYearEndOutput{}, fmt.Errorf("commit settle year end transaction: %w", err)
		}

		committed = true
	}

	return SettleYearEndOutput{
		Settlement: &settlement,
		Account:    &account,
	}, nil
}

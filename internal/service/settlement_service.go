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

var _ SettlementService = (*settlementService)(nil)

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

func (s *settlementService) SettleStudentLeaving(
	ctx context.Context,
	input SettleStudentLeavingInput,
) (SettleStudentLeavingOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleWaliKelas); err != nil {
		return SettleStudentLeavingOutput{}, err
	}

	if input.SavingsAccountID == uuid.Nil {
		return SettleStudentLeavingOutput{}, domain.ErrInvalidID
	}

	uow, repos, txCtx, ownsTransaction, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return SettleStudentLeavingOutput{}, fmt.Errorf("begin settle student leaving transaction: %w", err)
	}

	ctx = txCtx
	committed := false
	defer func() {
		if ownsTransaction && !committed {
			_ = uow.Rollback()
		}
	}()

	// Student-leaving settlement is also a financial write. Serialize it with
	// year lifecycle transitions using the same year-before-account order.
	unlocked, err := repos.SavingsAccounts.GetByID(ctx, input.SavingsAccountID)
	if err != nil {
		return SettleStudentLeavingOutput{}, err
	}
	year, err := repos.AcademicYears.GetByIDForShare(ctx, unlocked.AcademicYearID)
	if err != nil {
		return SettleStudentLeavingOutput{}, err
	}
	if year.IsClosed() {
		return SettleStudentLeavingOutput{}, domain.ErrInvalidTransition
	}
	account, err := repos.SavingsAccounts.GetByIDForUpdate(ctx, input.SavingsAccountID)
	if err != nil {
		return SettleStudentLeavingOutput{}, err
	}

	if err := s.authorizeAccount(ctx, repos, input.Actor, account); err != nil {
		return SettleStudentLeavingOutput{}, err
	}

	if !account.IsOpen() {
		return SettleStudentLeavingOutput{}, domain.ErrInvalidTransition
	}
	if pending, e := hasPendingLateRequest(ctx, repos, account.ID); e != nil {
		return SettleStudentLeavingOutput{}, e
	} else if pending {
		return SettleStudentLeavingOutput{}, domain.ErrInvalidTransition
	}

	student, err := repos.Students.GetByID(ctx, account.StudentID)
	if err != nil {
		return SettleStudentLeavingOutput{}, err
	}

	if !student.IsLeft() {
		return SettleStudentLeavingOutput{}, domain.ErrInvalidTransition
	}

	transactions, err := repos.Transactions.ListActiveBySavingsAccount(ctx, account.ID)
	if err != nil {
		return SettleStudentLeavingOutput{}, err
	}

	balance, err := domain.CalculateBalance(transactions)
	if err != nil {
		return SettleStudentLeavingOutput{}, err
	}
	if balance.Amount != input.ExpectedBalance {
		return SettleStudentLeavingOutput{}, repository.ErrConflict
	}

	now := time.Now()
	settlement, err := domain.NewSavingsSettlement(
		account.ID,
		domain.SettlementStudentLeaving,
		balance.Amount,
		input.Actor.UserID,
		now,
		nil,
	)
	if err != nil {
		return SettleStudentLeavingOutput{}, err
	}

	beforeAccount := account
	if err := account.Settle(settlement.ExecutedAt); err != nil {
		return SettleStudentLeavingOutput{}, err
	}

	if err := repos.SavingsAccounts.Update(ctx, account); err != nil {
		return SettleStudentLeavingOutput{}, err
	}

	if err := repos.Settlements.Create(ctx, settlement); err != nil {
		return SettleStudentLeavingOutput{}, err
	}

	if err := s.createSettlementAudit(ctx, repos, input.Actor, "SETTLE", settlement, nil); err != nil {
		return SettleStudentLeavingOutput{}, err
	}

	if err := s.createSavingsAccountAudit(ctx, repos, input.Actor, "SETTLE", account, &beforeAccount); err != nil {
		return SettleStudentLeavingOutput{}, err
	}

	if ownsTransaction {
		if err := uow.Commit(); err != nil {
			return SettleStudentLeavingOutput{}, fmt.Errorf("commit settle student leaving transaction: %w", err)
		}

		committed = true
	}

	return SettleStudentLeavingOutput{
		Settlement: &settlement,
		Account:    &account,
		Student:    &student,
	}, nil
}

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

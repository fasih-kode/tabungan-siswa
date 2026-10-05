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

func (s *settlementService) SettleYearEnd(
	ctx context.Context,
	input SettleYearEndInput,
) (SettleYearEndOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin, domain.RoleWaliKelas); err != nil {
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

	account, err := repos.SavingsAccounts.GetByIDForUpdate(ctx, input.SavingsAccountID)
	if err != nil {
		return SettleYearEndOutput{}, err
	}

	if err := s.authorizeAccount(ctx, repos, input.Actor, account); err != nil {
		return SettleYearEndOutput{}, err
	}

	if !account.IsOpen() {
		return SettleYearEndOutput{}, domain.ErrInvalidTransition
	}

	academicYear, err := repos.AcademicYears.GetByID(ctx, account.AcademicYearID)
	if err != nil {
		return SettleYearEndOutput{}, err
	}

	if !academicYear.IsClosing() {
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

	now := time.Now()
	settlement, err := domain.NewSavingsSettlement(
		account.ID,
		domain.SettlementRegularYearEnd,
		balance.Amount,
		input.Actor.UserID,
		now,
		nil,
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
	if err := RequireRole(input.Actor, domain.RoleAdmin, domain.RoleWaliKelas); err != nil {
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

func (s *settlementService) Get(
	ctx context.Context,
	input GetSettlementInput,
) (GetSettlementOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetSettlementOutput{}, err
	}

	if input.SettlementID == uuid.Nil {
		return GetSettlementOutput{}, domain.ErrInvalidID
	}

	settlement, err := s.deps.Repositories.Settlements.GetByID(ctx, input.SettlementID)
	if err != nil {
		return GetSettlementOutput{}, err
	}

	account, err := s.deps.Repositories.SavingsAccounts.GetByID(ctx, settlement.SavingsAccountID)
	if err != nil {
		return GetSettlementOutput{}, err
	}

	if err := s.authorizeAccount(ctx, s.deps.Repositories, input.Actor, account); err != nil {
		return GetSettlementOutput{}, err
	}

	return GetSettlementOutput{Settlement: &settlement}, nil
}

func (s *settlementService) List(
	ctx context.Context,
	input ListSettlementsInput,
) (ListSettlementsOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return ListSettlementsOutput{}, err
	}

	if input.SavingsAccountID == uuid.Nil {
		return ListSettlementsOutput{}, domain.ErrInvalidID
	}

	account, err := s.deps.Repositories.SavingsAccounts.GetByID(ctx, input.SavingsAccountID)
	if err != nil {
		return ListSettlementsOutput{}, err
	}

	if err := s.authorizeAccount(ctx, s.deps.Repositories, input.Actor, account); err != nil {
		return ListSettlementsOutput{}, err
	}

	settlements, err := s.deps.Repositories.Settlements.ListBySavingsAccount(
		ctx,
		account.ID,
		repository.ListOptions{
			Limit:  input.Options.Limit,
			Offset: input.Options.Offset,
		},
	)
	if err != nil {
		return ListSettlementsOutput{}, err
	}

	return ListSettlementsOutput{Settlements: settlementPointers(settlements)}, nil
}

func (s *settlementService) Reopen(
	ctx context.Context,
	input ReopenSettlementInput,
) (ReopenSettlementOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin); err != nil {
		return ReopenSettlementOutput{}, err
	}

	if input.SettlementID == uuid.Nil {
		return ReopenSettlementOutput{}, domain.ErrInvalidID
	}

	uow, repos, txCtx, ownsTransaction, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return ReopenSettlementOutput{}, fmt.Errorf("begin reopen settlement transaction: %w", err)
	}

	ctx = txCtx
	committed := false
	defer func() {
		if ownsTransaction && !committed {
			_ = uow.Rollback()
		}
	}()

	settlement, err := repos.Settlements.GetByID(ctx, input.SettlementID)
	if err != nil {
		return ReopenSettlementOutput{}, err
	}

	account, err := repos.SavingsAccounts.GetByIDForUpdate(ctx, settlement.SavingsAccountID)
	if err != nil {
		return ReopenSettlementOutput{}, err
	}

	if !account.IsSettled() {
		return ReopenSettlementOutput{}, domain.ErrInvalidTransition
	}

	current, err := repos.Settlements.GetCompletedBySavingsAccount(ctx, account.ID)
	if err != nil {
		return ReopenSettlementOutput{}, err
	}

	if current.ID != settlement.ID {
		return ReopenSettlementOutput{}, repository.ErrConflict
	}

	beforeSettlement := settlement
	if err := settlement.Supersede(); err != nil {
		return ReopenSettlementOutput{}, err
	}

	beforeAccount := account
	if err := account.Reopen(); err != nil {
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

	if ownsTransaction {
		if err := uow.Commit(); err != nil {
			return ReopenSettlementOutput{}, fmt.Errorf("commit reopen settlement transaction: %w", err)
		}

		committed = true
	}

	return ReopenSettlementOutput{
		PreviousSettlement: &settlement,
		Account:            &account,
	}, nil
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

func settlementPointers(settlements []domain.SavingsSettlement) []*domain.SavingsSettlement {
	result := make([]*domain.SavingsSettlement, len(settlements))
	for i := range settlements {
		settlement := settlements[i]
		result[i] = &settlement
	}
	return result
}

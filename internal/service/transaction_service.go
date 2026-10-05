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

var _ TransactionService = (*transactionService)(nil)

func (s *transactionService) Deposit(
	ctx context.Context,
	input DepositInput,
) (DepositOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin, domain.RoleWaliKelas); err != nil {
		return DepositOutput{}, err
	}

	if input.SavingsAccountID == uuid.Nil {
		return DepositOutput{}, ErrScopeViolation
	}

	uow, repos, txCtx, ownsTransaction, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return DepositOutput{}, fmt.Errorf("begin deposit transaction: %w", err)
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
		return DepositOutput{}, err
	}

	if err := s.authorizeAccount(ctx, repos, input.Actor, account); err != nil {
		return DepositOutput{}, err
	}

	if !account.IsOpen() {
		return DepositOutput{}, domain.ErrInvalidTransition
	}

	transaction, err := domain.NewDeposit(
		account.ID,
		input.Amount,
		input.TransactionDate,
		input.Actor.UserID,
	)
	if err != nil {
		return DepositOutput{}, err
	}

	if err := repos.Transactions.Create(ctx, transaction); err != nil {
		return DepositOutput{}, err
	}

	if err := s.createTransactionAudit(
		ctx,
		repos,
		input.Actor,
		"CREATE",
		transaction,
		nil,
	); err != nil {
		return DepositOutput{}, err
	}

	if ownsTransaction {
		if err := uow.Commit(); err != nil {
			return DepositOutput{}, fmt.Errorf("commit deposit transaction: %w", err)
		}

		committed = true
	}

	return DepositOutput{Transaction: &transaction}, nil
}

func (s *transactionService) Withdrawal(
	ctx context.Context,
	input WithdrawalInput,
) (WithdrawalOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin, domain.RoleWaliKelas); err != nil {
		return WithdrawalOutput{}, err
	}

	if input.SavingsAccountID == uuid.Nil {
		return WithdrawalOutput{}, ErrScopeViolation
	}

	uow, repos, txCtx, ownsTransaction, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return WithdrawalOutput{}, fmt.Errorf("begin withdrawal transaction: %w", err)
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
		return WithdrawalOutput{}, err
	}

	if err := s.authorizeAccount(ctx, repos, input.Actor, account); err != nil {
		return WithdrawalOutput{}, err
	}

	if !account.IsOpen() {
		return WithdrawalOutput{}, domain.ErrInvalidTransition
	}

	transactions, err := repos.Transactions.ListActiveBySavingsAccount(ctx, account.ID)
	if err != nil {
		return WithdrawalOutput{}, err
	}

	if _, err := domain.CalculateBalance(transactions); err != nil {
		return WithdrawalOutput{}, err
	}

	withdrawal, err := domain.NewWithdrawal(
		account.ID,
		input.Amount,
		input.TransactionDate,
		input.Actor.UserID,
	)
	if err != nil {
		return WithdrawalOutput{}, err
	}

	transactions = append(transactions, withdrawal)
	if _, err := domain.CalculateBalance(transactions); err != nil {
		return WithdrawalOutput{}, err
	}

	if err := repos.Transactions.Create(ctx, withdrawal); err != nil {
		return WithdrawalOutput{}, err
	}

	if err := s.createTransactionAudit(
		ctx,
		repos,
		input.Actor,
		"CREATE",
		withdrawal,
		nil,
	); err != nil {
		return WithdrawalOutput{}, err
	}

	if ownsTransaction {
		if err := uow.Commit(); err != nil {
			return WithdrawalOutput{}, fmt.Errorf("commit withdrawal transaction: %w", err)
		}

		committed = true
	}

	return WithdrawalOutput{Transaction: &withdrawal}, nil
}

func (s *transactionService) Get(
	ctx context.Context,
	input GetTransactionInput,
) (GetTransactionOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetTransactionOutput{}, err
	}

	if input.TransactionID == uuid.Nil {
		return GetTransactionOutput{}, ErrScopeViolation
	}

	transaction, err := s.deps.Repositories.Transactions.GetByID(ctx, input.TransactionID)
	if err != nil {
		return GetTransactionOutput{}, err
	}

	account, err := s.deps.Repositories.SavingsAccounts.GetByID(ctx, transaction.SavingsAccountID)
	if err != nil {
		return GetTransactionOutput{}, err
	}

	if err := s.authorizeAccount(ctx, s.deps.Repositories, input.Actor, account); err != nil {
		return GetTransactionOutput{}, err
	}

	return GetTransactionOutput{Transaction: &transaction}, nil
}

func (s *transactionService) List(
	ctx context.Context,
	input ListTransactionsInput,
) (ListTransactionsOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return ListTransactionsOutput{}, err
	}

	if input.SavingsAccountID == uuid.Nil {
		return ListTransactionsOutput{}, ErrScopeViolation
	}

	account, err := s.deps.Repositories.SavingsAccounts.GetByID(ctx, input.SavingsAccountID)
	if err != nil {
		return ListTransactionsOutput{}, err
	}

	if err := s.authorizeAccount(ctx, s.deps.Repositories, input.Actor, account); err != nil {
		return ListTransactionsOutput{}, err
	}

	transactions, err := s.deps.Repositories.Transactions.ListBySavingsAccount(
		ctx,
		account.ID,
		repository.ListOptions{
			Limit:  input.Options.Limit,
			Offset: input.Options.Offset,
		},
	)
	if err != nil {
		return ListTransactionsOutput{}, err
	}

	return ListTransactionsOutput{Transactions: transactionPointers(transactions)}, nil
}

func (s *transactionService) Edit(
	ctx context.Context,
	input EditTransactionInput,
) (EditTransactionOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin, domain.RoleWaliKelas); err != nil {
		return EditTransactionOutput{}, err
	}

	if input.TransactionID == uuid.Nil {
		return EditTransactionOutput{}, ErrScopeViolation
	}

	uow, repos, txCtx, ownsTransaction, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return EditTransactionOutput{}, fmt.Errorf("begin edit transaction: %w", err)
	}

	ctx = txCtx
	committed := false
	defer func() {
		if ownsTransaction && !committed {
			_ = uow.Rollback()
		}
	}()

	existing, err := repos.Transactions.GetByID(ctx, input.TransactionID)
	if err != nil {
		return EditTransactionOutput{}, err
	}

	account, err := repos.SavingsAccounts.GetByIDForUpdate(ctx, existing.SavingsAccountID)
	if err != nil {
		return EditTransactionOutput{}, err
	}

	if err := s.authorizeAccount(ctx, repos, input.Actor, account); err != nil {
		return EditTransactionOutput{}, err
	}

	if !account.IsOpen() {
		return EditTransactionOutput{}, domain.ErrInvalidTransition
	}

	if existing.Status != domain.TransactionActive {
		return EditTransactionOutput{}, domain.ErrInvalidTransition
	}

	// Re-read after acquiring the account lock so the mutation uses the
	// transaction state observed inside the same transaction boundary.
	existing, err = repos.Transactions.GetByID(ctx, input.TransactionID)
	if err != nil {
		return EditTransactionOutput{}, err
	}

	if existing.Status != domain.TransactionActive {
		return EditTransactionOutput{}, domain.ErrInvalidTransition
	}

	updated := existing
	updated.Amount = input.Amount
	updated.TransactionDate = input.TransactionDate
	updated.UpdatedAt = time.Now()

	transactions, err := repos.Transactions.ListActiveBySavingsAccount(ctx, account.ID)
	if err != nil {
		return EditTransactionOutput{}, err
	}

	found := false
	for i := range transactions {
		if transactions[i].ID == updated.ID {
			transactions[i] = updated
			found = true
			break
		}
	}
	if !found {
		return EditTransactionOutput{}, repository.ErrNotFound
	}

	if _, err := domain.CalculateBalance(transactions); err != nil {
		return EditTransactionOutput{}, err
	}

	if err := repos.Transactions.Update(ctx, updated); err != nil {
		return EditTransactionOutput{}, err
	}

	if err := s.createTransactionAudit(
		ctx,
		repos,
		input.Actor,
		"UPDATE",
		updated,
		&existing,
	); err != nil {
		return EditTransactionOutput{}, err
	}

	if ownsTransaction {
		if err := uow.Commit(); err != nil {
			return EditTransactionOutput{}, fmt.Errorf("commit edit transaction: %w", err)
		}

		committed = true
	}

	return EditTransactionOutput{Transaction: &updated}, nil
}

func (s *transactionService) Cancel(
	ctx context.Context,
	input CancelTransactionInput,
) (CancelTransactionOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleAdmin, domain.RoleWaliKelas); err != nil {
		return CancelTransactionOutput{}, err
	}

	if input.TransactionID == uuid.Nil {
		return CancelTransactionOutput{}, ErrScopeViolation
	}

	uow, repos, txCtx, ownsTransaction, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return CancelTransactionOutput{}, fmt.Errorf("begin cancel transaction: %w", err)
	}

	ctx = txCtx
	committed := false
	defer func() {
		if ownsTransaction && !committed {
			_ = uow.Rollback()
		}
	}()

	existing, err := repos.Transactions.GetByID(ctx, input.TransactionID)
	if err != nil {
		return CancelTransactionOutput{}, err
	}

	account, err := repos.SavingsAccounts.GetByIDForUpdate(ctx, existing.SavingsAccountID)
	if err != nil {
		return CancelTransactionOutput{}, err
	}

	if err := s.authorizeAccount(ctx, repos, input.Actor, account); err != nil {
		return CancelTransactionOutput{}, err
	}

	existing, err = repos.Transactions.GetByID(ctx, input.TransactionID)
	if err != nil {
		return CancelTransactionOutput{}, err
	}

	before := existing
	if err := existing.Cancel(); err != nil {
		return CancelTransactionOutput{}, err
	}

	if err := repos.Transactions.Update(ctx, existing); err != nil {
		return CancelTransactionOutput{}, err
	}

	if err := s.createTransactionAudit(
		ctx,
		repos,
		input.Actor,
		"CANCEL",
		existing,
		&before,
	); err != nil {
		return CancelTransactionOutput{}, err
	}

	if ownsTransaction {
		if err := uow.Commit(); err != nil {
			return CancelTransactionOutput{}, fmt.Errorf("commit cancel transaction: %w", err)
		}

		committed = true
	}

	return CancelTransactionOutput{Transaction: &existing}, nil
}

func (s *transactionService) authorizeAccount(
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

func (s *transactionService) createTransactionAudit(
	ctx context.Context,
	repos repository.RepositorySet,
	actor Actor,
	action string,
	after domain.Transaction,
	before *domain.Transaction,
) error {
	var beforeData []byte
	var err error
	if before != nil {
		beforeData, err = json.Marshal(before)
		if err != nil {
			return fmt.Errorf("marshal transaction audit before data: %w", err)
		}
	}

	afterData, err := json.Marshal(after)
	if err != nil {
		return fmt.Errorf("marshal transaction audit after data: %w", err)
	}

	_, err = recordAudit(ctx, repos, RecordAuditInput{
		Actor:      actor,
		Action:     action,
		EntityType: domain.AuditEntityTransaction,
		EntityID:   after.ID,
		Before:     beforeData,
		After:      afterData,
	})
	return err
}

func transactionPointers(transactions []domain.Transaction) []*domain.Transaction {
	result := make([]*domain.Transaction, len(transactions))
	for i := range transactions {
		transaction := transactions[i]
		result[i] = &transaction
	}
	return result
}

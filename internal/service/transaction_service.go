package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/fasih/tabungan-siswa/internal/repository"
	"github.com/google/uuid"
)

var _ TransactionService = (*transactionService)(nil)

func (s *transactionService) CanCorrect(ctx context.Context, actor Actor, transactionID uuid.UUID) (bool, error) {
	if actor.Role != domain.RoleWaliKelas || transactionID == uuid.Nil {
		return false, nil
	}
	tx, err := s.deps.Repositories.Transactions.GetByID(ctx, transactionID)
	if err != nil {
		return false, err
	}
	if tx.Status != domain.TransactionActive {
		return false, nil
	}
	account, err := s.deps.Repositories.SavingsAccounts.GetByID(ctx, tx.SavingsAccountID)
	if err != nil {
		return false, err
	}
	if !account.IsOpen() {
		return false, nil
	}
	year, err := s.deps.Repositories.AcademicYears.GetByID(ctx, account.AcademicYearID)
	if err != nil {
		return false, err
	}
	if year.IsOpen() {
		if err := s.authorizeAccount(ctx, s.deps.Repositories, actor, account); err != nil {
			return false, nil
		}
		return true, nil
	}
	if year.IsClosing() {
		_, err := correctionContextForApprovedRequest(ctx, s.deps.Repositories, actor, year, account.ID)
		if err == nil {
			return true, nil
		}
		if errors.Is(err, domain.ErrInvalidTransition) || errors.Is(err, ErrScopeViolation) {
			return false, nil
		}
		return false, err
	}
	return false, nil
}

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

	account, err := lockOpenTransactionAccount(ctx, repos, input.SavingsAccountID, input.TransactionDate)
	if err != nil {
		return DepositOutput{}, err
	}

	if err := s.authorizeAccount(ctx, repos, input.Actor, account); err != nil {
		return DepositOutput{}, err
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

	transactions, err := repos.Transactions.ListActiveBySavingsAccount(ctx, account.ID)
	if err != nil {
		return DepositOutput{}, err
	}
	transactions = append(transactions, transaction)
	sortTransactionsChronologically(transactions)
	if _, err := domain.CalculateBalance(transactions); err != nil {
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

	account, err := lockOpenTransactionAccount(ctx, repos, input.SavingsAccountID, input.TransactionDate)
	if err != nil {
		return WithdrawalOutput{}, err
	}

	if err := s.authorizeAccount(ctx, repos, input.Actor, account); err != nil {
		return WithdrawalOutput{}, err
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
	sortTransactionsChronologically(transactions)
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

	allTransactions, err := s.deps.Repositories.Transactions.ListAllBySavingsAccount(ctx, account.ID)
	if err != nil {
		return ListTransactionsOutput{}, err
	}
	transactions := make([]domain.Transaction, 0, len(allTransactions))
	for _, transaction := range allTransactions {
		if transaction.Status == domain.TransactionActive {
			transactions = append(transactions, transaction)
		}
	}
	sortTransactionsChronologically(transactions)
	runningBalances := make(map[uuid.UUID]domain.Money, len(transactions))
	var running domain.Money
	for _, transaction := range transactions {
		if !transaction.Amount.IsPositive() || transaction.Amount > domain.MaxMoney {
			return ListTransactionsOutput{}, domain.ErrInvalidAmount
		}
		switch transaction.Type {
		case domain.TransactionDeposit:
			running, err = running.Add(transaction.Amount)
		case domain.TransactionWithdrawal:
			running, err = running.Subtract(transaction.Amount)
		default:
			return ListTransactionsOutput{}, domain.ErrInvalidValue
		}
		if err != nil {
			return ListTransactionsOutput{}, err
		}
		runningBalances[transaction.ID] = running
	}
	sortTransactionsChronologically(allTransactions)

	start := input.Options.Offset
	if start < 0 {
		return ListTransactionsOutput{}, domain.ErrInvalidValue
	}
	limit := input.Options.Limit
	if limit <= 0 {
		limit = 10
	}
	if start > len(allTransactions) {
		start = len(allTransactions)
	}
	end := start + limit
	if end < start || end > len(allTransactions) {
		end = len(allTransactions)
	}
	page := allTransactions[start:end]
	return ListTransactionsOutput{
		Transactions:    transactionPointers(page),
		RunningBalances: runningBalances,
		Balance:         running,
		Total:           len(allTransactions),
	}, nil
}

func (s *transactionService) Edit(
	ctx context.Context,
	input EditTransactionInput,
) (EditTransactionOutput, error) {
	if err := RequireRole(input.Actor, domain.RoleWaliKelas); err != nil {
		return EditTransactionOutput{}, err
	}

	if input.TransactionID == uuid.Nil {
		return EditTransactionOutput{}, ErrScopeViolation
	}
	if !input.Amount.IsPositive() || input.Amount > domain.MaxMoney {
		return EditTransactionOutput{}, domain.ErrInvalidAmount
	}
	reason, err := validatedCorrectionReason(input.Reason)
	if err != nil {
		return EditTransactionOutput{}, err
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
	unlockedAccount, err := repos.SavingsAccounts.GetByID(ctx, existing.SavingsAccountID)
	if err != nil {
		return EditTransactionOutput{}, err
	}
	year, err := repos.AcademicYears.GetByIDForShare(ctx, unlockedAccount.AcademicYearID)
	if err != nil {
		return EditTransactionOutput{}, err
	}
	if !transactionDateWithinAcademicYear(input.TransactionDate, year) {
		return EditTransactionOutput{}, ErrTransactionDateOutOfAcademicYear
	}
	ctx, err = correctionContextForApprovedRequest(ctx, repos, input.Actor, year, unlockedAccount.ID)
	if err != nil {
		return EditTransactionOutput{}, err
	}

	account, err := repos.SavingsAccounts.GetByIDForUpdate(ctx, existing.SavingsAccountID)
	if err != nil {
		return EditTransactionOutput{}, err
	}
	if account.AcademicYearID != year.ID {
		return EditTransactionOutput{}, domain.ErrInvalidTransition
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
	sortTransactionsChronologically(transactions)

	if _, err := domain.CalculateBalance(transactions); err != nil {
		return EditTransactionOutput{}, err
	}

	if err := repos.Transactions.Update(ctx, updated); err != nil {
		return EditTransactionOutput{}, err
	}

	if err := s.createCorrectionAudit(
		ctx,
		repos,
		input.Actor,
		"UPDATE",
		updated,
		existing,
		reason,
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
	if err := RequireRole(input.Actor, domain.RoleWaliKelas); err != nil {
		return CancelTransactionOutput{}, err
	}

	if input.TransactionID == uuid.Nil {
		return CancelTransactionOutput{}, ErrScopeViolation
	}
	reason, err := validatedCorrectionReason(input.Reason)
	if err != nil {
		return CancelTransactionOutput{}, err
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

	unlockedAccount, err := repos.SavingsAccounts.GetByID(ctx, existing.SavingsAccountID)
	if err != nil {
		return CancelTransactionOutput{}, err
	}
	year, err := repos.AcademicYears.GetByIDForShare(ctx, unlockedAccount.AcademicYearID)
	if err != nil {
		return CancelTransactionOutput{}, err
	}
	ctx, err = correctionContextForApprovedRequest(ctx, repos, input.Actor, year, unlockedAccount.ID)
	if err != nil {
		return CancelTransactionOutput{}, err
	}
	account, err := repos.SavingsAccounts.GetByIDForUpdate(ctx, existing.SavingsAccountID)
	if err != nil {
		return CancelTransactionOutput{}, err
	}
	if account.AcademicYearID != year.ID {
		return CancelTransactionOutput{}, domain.ErrInvalidTransition
	}

	if err := s.authorizeAccount(ctx, repos, input.Actor, account); err != nil {
		return CancelTransactionOutput{}, err
	}
	if !account.IsOpen() {
		return CancelTransactionOutput{}, domain.ErrInvalidTransition
	}

	existing, err = repos.Transactions.GetByID(ctx, input.TransactionID)
	if err != nil {
		return CancelTransactionOutput{}, err
	}
	if !transactionDateWithinAcademicYear(existing.TransactionDate, year) {
		return CancelTransactionOutput{}, ErrTransactionDateOutOfAcademicYear
	}

	before := existing
	if existing.Status != domain.TransactionActive {
		return CancelTransactionOutput{}, domain.ErrAlreadyCancelled
	}
	transactions, err := repos.Transactions.ListActiveBySavingsAccount(ctx, account.ID)
	if err != nil {
		return CancelTransactionOutput{}, err
	}
	if err := existing.Cancel(); err != nil {
		return CancelTransactionOutput{}, err
	}
	found := false
	for i := range transactions {
		if transactions[i].ID == existing.ID {
			transactions[i] = existing
			found = true
			break
		}
	}
	if !found {
		transactions = append(transactions, existing)
	}
	sortTransactionsChronologically(transactions)
	if _, err := domain.CalculateBalance(transactions); err != nil {
		return CancelTransactionOutput{}, err
	}

	if err := repos.Transactions.Update(ctx, existing); err != nil {
		return CancelTransactionOutput{}, err
	}

	if err := s.createCorrectionAudit(
		ctx,
		repos,
		input.Actor,
		"CANCEL",
		existing,
		before,
		reason,
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

func validatedCorrectionReason(raw string) (string, error) {
	reason := strings.TrimSpace(raw)
	if reason == "" || utf8.RuneCountInString(reason) > 500 {
		return "", domain.ErrInvalidValue
	}
	return reason, nil
}

func correctionYearAllowed(ctx context.Context, year domain.AcademicYear, accountID uuid.UUID) bool {
	if year.IsOpen() {
		return true
	}
	if !year.IsClosing() {
		return false
	}
	scope, ok := approvedCorrectionScopeFromContext(ctx)
	return ok && scope.AccountID == accountID
}

func (s *transactionService) createCorrectionAudit(
	ctx context.Context,
	repos repository.RepositorySet,
	actor Actor,
	action string,
	after domain.Transaction,
	before domain.Transaction,
	reason string,
) error {
	beforeData, err := json.Marshal(before)
	if err != nil {
		return fmt.Errorf("marshal transaction audit before data: %w", err)
	}
	afterData, err := json.Marshal(struct {
		Transaction          domain.Transaction `json:"transaction"`
		Reason               string             `json:"reason"`
		ApprovedCorrectionID *uuid.UUID         `json:"approved_correction_id,omitempty"`
	}{Transaction: after, Reason: reason, ApprovedCorrectionID: approvedCorrectionID(ctx)})
	if err != nil {
		return fmt.Errorf("marshal transaction audit after data: %w", err)
	}
	_, err = recordAudit(ctx, repos, RecordAuditInput{
		Actor: actor, Action: action, EntityType: domain.AuditEntityTransaction,
		EntityID: after.ID, Before: beforeData, After: afterData,
	})
	return err
}

func approvedCorrectionID(ctx context.Context) *uuid.UUID {
	scope, ok := approvedCorrectionScopeFromContext(ctx)
	if !ok {
		return nil
	}
	id := scope.RequestID
	return &id
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

func sortTransactionsChronologically(transactions []domain.Transaction) {
	sort.SliceStable(transactions, func(i, j int) bool {
		leftDate := transactions[i].TransactionDate
		rightDate := transactions[j].TransactionDate
		if leftDate.Year() != rightDate.Year() || leftDate.Month() != rightDate.Month() || leftDate.Day() != rightDate.Day() {
			leftDay := time.Date(leftDate.Year(), leftDate.Month(), leftDate.Day(), 0, 0, 0, 0, time.UTC)
			rightDay := time.Date(rightDate.Year(), rightDate.Month(), rightDate.Day(), 0, 0, 0, 0, time.UTC)
			return leftDay.Before(rightDay)
		}
		if !transactions[i].CreatedAt.Equal(transactions[j].CreatedAt) {
			return transactions[i].CreatedAt.Before(transactions[j].CreatedAt)
		}
		return transactions[i].ID.String() < transactions[j].ID.String()
	})
}

func lockOpenTransactionAccount(
	ctx context.Context,
	repos repository.RepositorySet,
	accountID uuid.UUID,
	transactionDate time.Time,
) (domain.SavingsAccount, error) {
	account, err := repos.SavingsAccounts.GetByID(ctx, accountID)
	if err != nil {
		return domain.SavingsAccount{}, err
	}
	if account.AcademicYearID == uuid.Nil {
		return domain.SavingsAccount{}, domain.ErrInvalidID
	}

	year, err := repos.AcademicYears.GetByIDForShare(ctx, account.AcademicYearID)
	if err != nil {
		return domain.SavingsAccount{}, err
	}
	if !year.IsOpen() {
		return domain.SavingsAccount{}, domain.ErrInvalidTransition
	}
	if !transactionDateWithinAcademicYear(transactionDate, year) {
		return domain.SavingsAccount{}, ErrTransactionDateOutOfAcademicYear
	}

	account, err = repos.SavingsAccounts.GetByIDForUpdate(ctx, accountID)
	if err != nil {
		return domain.SavingsAccount{}, err
	}
	if account.AcademicYearID != year.ID {
		return domain.SavingsAccount{}, domain.ErrInvalidTransition
	}
	if !account.IsOpen() {
		return domain.SavingsAccount{}, domain.ErrInvalidTransition
	}
	return account, nil
}

func calendarDate(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func transactionDateWithinAcademicYear(date time.Time, year domain.AcademicYear) bool {
	date = calendarDate(date)
	start := calendarDate(year.StartDate)
	end := calendarDate(year.EndDate)
	return !date.Before(start) && date.Before(end)
}

func transactionPointers(transactions []domain.Transaction) []*domain.Transaction {
	result := make([]*domain.Transaction, len(transactions))
	for i := range transactions {
		transaction := transactions[i]
		result[i] = &transaction
	}
	return result
}

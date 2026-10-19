package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/google/uuid"
)

var _ SavingsAccountService = (*savingsAccountService)(nil)

func (s *savingsAccountService) Create(
	ctx context.Context,
	input CreateSavingsAccountInput,
) (CreateSavingsAccountOutput, error) {
	if err := RequireRole(
		input.Actor,
		domain.RoleAdmin,
		domain.RoleWaliKelas,
	); err != nil {
		return CreateSavingsAccountOutput{}, err
	}

	if input.StudentID == uuid.Nil || input.AcademicYearID == uuid.Nil {
		return CreateSavingsAccountOutput{}, ErrScopeViolation
	}

	if input.Actor.Role == domain.RoleWaliKelas {
		if err := RequireStudentScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			input.StudentID,
			input.AcademicYearID,
		); err != nil {
			return CreateSavingsAccountOutput{}, err
		}
	}

	if s.deps.Repositories.Students != nil {
		student, err := s.deps.Repositories.Students.GetByID(ctx, input.StudentID)
		if err != nil {
			return CreateSavingsAccountOutput{}, err
		}
		if !student.IsActive() {
			return CreateSavingsAccountOutput{}, domain.ErrInvalidTransition
		}
	}
	if s.deps.Repositories.AcademicYears != nil {
		academicYear, err := s.deps.Repositories.AcademicYears.GetByID(ctx, input.AcademicYearID)
		if err != nil {
			return CreateSavingsAccountOutput{}, err
		}
		if !academicYear.IsOpen() {
			return CreateSavingsAccountOutput{}, domain.ErrInvalidTransition
		}
	}

	account, err := domain.NewSavingsAccount(
		input.StudentID,
		input.AcademicYearID,
	)
	if err != nil {
		return CreateSavingsAccountOutput{}, err
	}

	uow, repos, txCtx, ownsTransaction, err := beginTransaction(ctx, s.deps.UOW)
	if err != nil {
		return CreateSavingsAccountOutput{}, fmt.Errorf(
			"begin create savings account transaction: %w",
			err,
		)
	}

	ctx = txCtx
	committed := false
	defer func() {
		if ownsTransaction && !committed {
			_ = uow.Rollback()
		}
	}()

	if err := repos.SavingsAccounts.Create(ctx, account); err != nil {
		return CreateSavingsAccountOutput{}, err
	}

	afterData, err := json.Marshal(account)
	if err != nil {
		return CreateSavingsAccountOutput{}, fmt.Errorf(
			"marshal created savings account audit data: %w",
			err,
		)
	}

	if _, err := recordAudit(ctx, repos, RecordAuditInput{
		Actor:      input.Actor,
		Action:     "CREATE",
		EntityType: domain.AuditEntitySavingsAccount,
		EntityID:   account.ID,
		After:      afterData,
	}); err != nil {
		return CreateSavingsAccountOutput{}, err
	}

	if ownsTransaction {
		if err := uow.Commit(); err != nil {
			return CreateSavingsAccountOutput{}, fmt.Errorf(
				"commit create savings account transaction: %w",
				err,
			)
		}

		committed = true
	}

	return CreateSavingsAccountOutput{
		Account: &account,
	}, nil
}

func (s *savingsAccountService) Get(
	ctx context.Context,
	input GetSavingsAccountInput,
) (GetSavingsAccountOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetSavingsAccountOutput{}, err
	}

	if input.SavingsAccountID == uuid.Nil {
		return GetSavingsAccountOutput{}, ErrScopeViolation
	}

	account, err := s.deps.Repositories.SavingsAccounts.GetByID(
		ctx,
		input.SavingsAccountID,
	)
	if err != nil {
		return GetSavingsAccountOutput{}, err
	}

	switch input.Actor.Role {
	case domain.RoleAdmin:
		return GetSavingsAccountOutput{
			Account: &account,
		}, nil

	case domain.RoleWaliKelas:
		if err := RequireStudentScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			account.StudentID,
			account.AcademicYearID,
		); err != nil {
			return GetSavingsAccountOutput{}, err
		}

		return GetSavingsAccountOutput{
			Account: &account,
		}, nil

	case domain.RoleSiswa:
		student, err := s.deps.Repositories.Students.GetByID(
			ctx,
			account.StudentID,
		)
		if err != nil {
			return GetSavingsAccountOutput{}, err
		}

		if student.UserID == nil {
			return GetSavingsAccountOutput{}, ErrScopeViolation
		}

		if err := RequireSelf(input.Actor, *student.UserID); err != nil {
			return GetSavingsAccountOutput{}, err
		}

		return GetSavingsAccountOutput{
			Account: &account,
		}, nil

	default:
		return GetSavingsAccountOutput{}, ErrForbidden
	}
}

func (s *savingsAccountService) GetPassbook(
	ctx context.Context,
	input GetSavingsAccountPassbookInput,
) (GetSavingsAccountPassbookOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetSavingsAccountPassbookOutput{}, err
	}
	if input.SavingsAccountID == uuid.Nil {
		return GetSavingsAccountPassbookOutput{}, ErrScopeViolation
	}

	account, err := s.deps.Repositories.SavingsAccounts.GetByID(ctx, input.SavingsAccountID)
	if err != nil {
		return GetSavingsAccountPassbookOutput{}, err
	}
	switch input.Actor.Role {
	case domain.RoleAdmin:
	case domain.RoleWaliKelas:
		if err := RequireStudentScope(ctx, s.deps.Repositories, input.Actor, account.StudentID, account.AcademicYearID); err != nil {
			return GetSavingsAccountPassbookOutput{}, err
		}
	case domain.RoleSiswa:
		student, err := s.deps.Repositories.Students.GetByID(ctx, account.StudentID)
		if err != nil {
			return GetSavingsAccountPassbookOutput{}, err
		}
		if student.UserID == nil {
			return GetSavingsAccountPassbookOutput{}, ErrScopeViolation
		}
		if err := RequireSelf(input.Actor, *student.UserID); err != nil {
			return GetSavingsAccountPassbookOutput{}, err
		}
	default:
		return GetSavingsAccountPassbookOutput{}, ErrForbidden
	}

	transactions, err := s.deps.Repositories.Transactions.ListActiveBySavingsAccount(ctx, account.ID)
	if err != nil {
		return GetSavingsAccountPassbookOutput{}, err
	}
	sortTransactionsChronologically(transactions)
	entries := make([]*SavingsPassbookEntry, 0, len(transactions))
	var balance domain.Money
	for i := range transactions {
		tx := transactions[i]
		if !tx.Amount.IsPositive() || tx.Amount > domain.MaxMoney {
			return GetSavingsAccountPassbookOutput{}, domain.ErrInvalidAmount
		}
		switch tx.Type {
		case domain.TransactionDeposit:
			balance, err = balance.Add(tx.Amount)
		case domain.TransactionWithdrawal:
			balance, err = balance.Subtract(tx.Amount)
		default:
			return GetSavingsAccountPassbookOutput{}, domain.ErrInvalidValue
		}
		if err != nil {
			return GetSavingsAccountPassbookOutput{}, err
		}
		if balance < 0 {
			return GetSavingsAccountPassbookOutput{}, domain.ErrNegativeBalance
		}
		txCopy := tx
		entries = append(entries, &SavingsPassbookEntry{Transaction: &txCopy, RunningBalance: balance})
	}
	return GetSavingsAccountPassbookOutput{Entries: entries, Balance: balance}, nil
}

func (s *savingsAccountService) GetByStudentAndAcademicYear(
	ctx context.Context,
	input GetSavingsAccountByStudentAndAcademicYearInput,
) (GetSavingsAccountByStudentAndAcademicYearOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetSavingsAccountByStudentAndAcademicYearOutput{}, err
	}

	if input.StudentID == uuid.Nil || input.AcademicYearID == uuid.Nil {
		return GetSavingsAccountByStudentAndAcademicYearOutput{}, ErrScopeViolation
	}

	switch input.Actor.Role {
	case domain.RoleAdmin:
	case domain.RoleWaliKelas:
		if err := RequireStudentScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			input.StudentID,
			input.AcademicYearID,
		); err != nil {
			return GetSavingsAccountByStudentAndAcademicYearOutput{}, err
		}

	case domain.RoleSiswa:
		student, err := s.deps.Repositories.Students.GetByID(
			ctx,
			input.StudentID,
		)
		if err != nil {
			return GetSavingsAccountByStudentAndAcademicYearOutput{}, err
		}

		if student.UserID == nil {
			return GetSavingsAccountByStudentAndAcademicYearOutput{}, ErrScopeViolation
		}

		if err := RequireSelf(input.Actor, *student.UserID); err != nil {
			return GetSavingsAccountByStudentAndAcademicYearOutput{}, err
		}

	default:
		return GetSavingsAccountByStudentAndAcademicYearOutput{}, ErrForbidden
	}

	account, err := s.deps.Repositories.SavingsAccounts.GetByStudentAndAcademicYear(
		ctx,
		input.StudentID,
		input.AcademicYearID,
	)
	if err != nil {
		return GetSavingsAccountByStudentAndAcademicYearOutput{}, err
	}

	return GetSavingsAccountByStudentAndAcademicYearOutput{
		Account: &account,
	}, nil
}

func (s *savingsAccountService) GetBalance(
	ctx context.Context,
	input GetSavingsAccountBalanceInput,
) (GetSavingsAccountBalanceOutput, error) {
	if err := input.Actor.Validate(); err != nil {
		return GetSavingsAccountBalanceOutput{}, err
	}

	if input.SavingsAccountID == uuid.Nil {
		return GetSavingsAccountBalanceOutput{}, ErrScopeViolation
	}

	account, err := s.deps.Repositories.SavingsAccounts.GetByID(
		ctx,
		input.SavingsAccountID,
	)
	if err != nil {
		return GetSavingsAccountBalanceOutput{}, err
	}

	switch input.Actor.Role {
	case domain.RoleAdmin:
	case domain.RoleWaliKelas:
		if err := RequireStudentScope(
			ctx,
			s.deps.Repositories,
			input.Actor,
			account.StudentID,
			account.AcademicYearID,
		); err != nil {
			return GetSavingsAccountBalanceOutput{}, err
		}

	case domain.RoleSiswa:
		student, err := s.deps.Repositories.Students.GetByID(
			ctx,
			account.StudentID,
		)
		if err != nil {
			return GetSavingsAccountBalanceOutput{}, err
		}

		if student.UserID == nil {
			return GetSavingsAccountBalanceOutput{}, ErrScopeViolation
		}

		if err := RequireSelf(input.Actor, *student.UserID); err != nil {
			return GetSavingsAccountBalanceOutput{}, err
		}

	default:
		return GetSavingsAccountBalanceOutput{}, ErrForbidden
	}

	transactions, err := s.deps.Repositories.Transactions.ListActiveBySavingsAccount(
		ctx,
		account.ID,
	)
	if err != nil {
		return GetSavingsAccountBalanceOutput{}, err
	}
	sortTransactionsChronologically(transactions)

	balance, err := domain.CalculateBalance(transactions)
	if err != nil {
		return GetSavingsAccountBalanceOutput{}, err
	}

	return GetSavingsAccountBalanceOutput{
		Balance: balance.Amount,
	}, nil
}

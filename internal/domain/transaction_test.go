package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewTransaction(t *testing.T) {
	savingsAccountID := uuid.New()
	createdBy := uuid.New()
	transactionDate := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)

	transaction, err := NewTransaction(
		savingsAccountID,
		TransactionDeposit,
		100_000,
		transactionDate,
		createdBy,
	)
	if err != nil {
		t.Fatalf("NewTransaction() error = %v", err)
	}

	if transaction.ID == uuid.Nil {
		t.Fatal("NewTransaction() ID must not be nil")
	}

	if transaction.SavingsAccountID != savingsAccountID {
		t.Fatalf(
			"NewTransaction() SavingsAccountID = %v, want %v",
			transaction.SavingsAccountID,
			savingsAccountID,
		)
	}

	if transaction.Type != TransactionDeposit {
		t.Fatalf(
			"NewTransaction() Type = %v, want %v",
			transaction.Type,
			TransactionDeposit,
		)
	}

	if transaction.Amount != 100_000 {
		t.Fatalf(
			"NewTransaction() Amount = %v, want %v",
			transaction.Amount,
			100_000,
		)
	}

	if !transaction.TransactionDate.Equal(transactionDate) {
		t.Fatalf(
			"NewTransaction() TransactionDate = %v, want %v",
			transaction.TransactionDate,
			transactionDate,
		)
	}

	if transaction.Status != TransactionActive {
		t.Fatalf(
			"NewTransaction() Status = %v, want %v",
			transaction.Status,
			TransactionActive,
		)
	}

	if transaction.CreatedBy != createdBy {
		t.Fatalf(
			"NewTransaction() CreatedBy = %v, want %v",
			transaction.CreatedBy,
			createdBy,
		)
	}

	if transaction.CreatedAt.IsZero() {
		t.Fatal("NewTransaction() CreatedAt must not be zero")
	}

	if transaction.UpdatedAt.IsZero() {
		t.Fatal("NewTransaction() UpdatedAt must not be zero")
	}
}

func TestNewTransactionRejectsNilSavingsAccountID(t *testing.T) {
	_, err := NewTransaction(
		uuid.Nil,
		TransactionDeposit,
		100_000,
		time.Now(),
		uuid.New(),
	)

	if err != ErrInvalidID {
		t.Fatalf(
			"NewTransaction() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}
}

func TestNewTransactionRejectsNilCreatedBy(t *testing.T) {
	_, err := NewTransaction(
		uuid.New(),
		TransactionDeposit,
		100_000,
		time.Now(),
		uuid.Nil,
	)

	if err != ErrInvalidID {
		t.Fatalf(
			"NewTransaction() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}
}

func TestNewTransactionRejectsInvalidType(t *testing.T) {
	_, err := NewTransaction(
		uuid.New(),
		TransactionType("INVALID"),
		100_000,
		time.Now(),
		uuid.New(),
	)

	if err != ErrInvalidValue {
		t.Fatalf(
			"NewTransaction() error = %v, want %v",
			err,
			ErrInvalidValue,
		)
	}
}

func TestNewTransactionRejectsZeroAmount(t *testing.T) {
	_, err := NewTransaction(
		uuid.New(),
		TransactionDeposit,
		0,
		time.Now(),
		uuid.New(),
	)

	if err != ErrInvalidAmount {
		t.Fatalf(
			"NewTransaction() error = %v, want %v",
			err,
			ErrInvalidAmount,
		)
	}
}

func TestNewTransactionRejectsNegativeAmount(t *testing.T) {
	_, err := NewTransaction(
		uuid.New(),
		TransactionDeposit,
		-1,
		time.Now(),
		uuid.New(),
	)

	if err != ErrInvalidAmount {
		t.Fatalf(
			"NewTransaction() error = %v, want %v",
			err,
			ErrInvalidAmount,
		)
	}
}

func TestNewDeposit(t *testing.T) {
	transactionDate := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	savingsAccountID := uuid.New()
	createdBy := uuid.New()

	transaction, err := NewDeposit(
		savingsAccountID,
		100_000,
		transactionDate,
		createdBy,
	)
	if err != nil {
		t.Fatalf("NewDeposit() error = %v", err)
	}

	if transaction.Type != TransactionDeposit {
		t.Fatalf(
			"NewDeposit() Type = %v, want %v",
			transaction.Type,
			TransactionDeposit,
		)
	}

	if transaction.Amount != 100_000 {
		t.Fatalf(
			"NewDeposit() Amount = %v, want %v",
			transaction.Amount,
			100_000,
		)
	}

	if transaction.SavingsAccountID != savingsAccountID {
		t.Fatalf(
			"NewDeposit() SavingsAccountID = %v, want %v",
			transaction.SavingsAccountID,
			savingsAccountID,
		)
	}

	if transaction.CreatedBy != createdBy {
		t.Fatalf(
			"NewDeposit() CreatedBy = %v, want %v",
			transaction.CreatedBy,
			createdBy,
		)
	}
}

func TestNewWithdrawal(t *testing.T) {
	transactionDate := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	savingsAccountID := uuid.New()
	createdBy := uuid.New()

	transaction, err := NewWithdrawal(
		savingsAccountID,
		75_000,
		transactionDate,
		createdBy,
	)
	if err != nil {
		t.Fatalf("NewWithdrawal() error = %v", err)
	}

	if transaction.Type != TransactionWithdrawal {
		t.Fatalf(
			"NewWithdrawal() Type = %v, want %v",
			transaction.Type,
			TransactionWithdrawal,
		)
	}

	if transaction.Amount != 75_000 {
		t.Fatalf(
			"NewWithdrawal() Amount = %v, want %v",
			transaction.Amount,
			75_000,
		)
	}

	if transaction.SavingsAccountID != savingsAccountID {
		t.Fatalf(
			"NewWithdrawal() SavingsAccountID = %v, want %v",
			transaction.SavingsAccountID,
			savingsAccountID,
		)
	}

	if transaction.CreatedBy != createdBy {
		t.Fatalf(
			"NewWithdrawal() CreatedBy = %v, want %v",
			transaction.CreatedBy,
			createdBy,
		)
	}
}

func TestTransactionIsActive(t *testing.T) {
	transaction, err := NewDeposit(
		uuid.New(),
		100_000,
		time.Now(),
		uuid.New(),
	)
	if err != nil {
		t.Fatalf("NewDeposit() error = %v", err)
	}

	if !transaction.IsActive() {
		t.Fatal("Transaction.IsActive() = false, want true")
	}

	if transaction.IsCancelled() {
		t.Fatal("Transaction.IsCancelled() = true, want false")
	}
}

func TestTransactionCancel(t *testing.T) {
	transaction, err := NewDeposit(
		uuid.New(),
		100_000,
		time.Now(),
		uuid.New(),
	)
	if err != nil {
		t.Fatalf("NewDeposit() error = %v", err)
	}

	if err := transaction.Cancel(); err != nil {
		t.Fatalf("Transaction.Cancel() error = %v", err)
	}

	if transaction.Status != TransactionCancelled {
		t.Fatalf(
			"Transaction.Cancel() Status = %v, want %v",
			transaction.Status,
			TransactionCancelled,
		)
	}

	if transaction.IsActive() {
		t.Fatal("Transaction.IsActive() = true after cancellation, want false")
	}

	if !transaction.IsCancelled() {
		t.Fatal("Transaction.IsCancelled() = false after cancellation, want true")
	}
}

func TestTransactionCannotCancelTwice(t *testing.T) {
	transaction, err := NewDeposit(
		uuid.New(),
		100_000,
		time.Now(),
		uuid.New(),
	)
	if err != nil {
		t.Fatalf("NewDeposit() error = %v", err)
	}

	if err := transaction.Cancel(); err != nil {
		t.Fatalf("Transaction.Cancel() first call error = %v", err)
	}

	if err := transaction.Cancel(); err != ErrAlreadyCancelled {
		t.Fatalf(
			"Transaction.Cancel() second call error = %v, want %v",
			err,
			ErrAlreadyCancelled,
		)
	}
}

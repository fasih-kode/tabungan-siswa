package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCalculateBalance(t *testing.T) {
	accountID := uuid.New()
	userID := uuid.New()
	date := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	deposit1, err := NewDeposit(accountID, 100000, date, userID)
	if err != nil {
		t.Fatalf("NewDeposit() unexpected error: %v", err)
	}

	deposit2, err := NewDeposit(accountID, 50000, date, userID)
	if err != nil {
		t.Fatalf("NewDeposit() unexpected error: %v", err)
	}

	withdrawal, err := NewWithdrawal(accountID, 25000, date, userID)
	if err != nil {
		t.Fatalf("NewWithdrawal() unexpected error: %v", err)
	}

	got, err := CalculateBalance([]Transaction{deposit1, deposit2, withdrawal})
	if err != nil {
		t.Fatalf("CalculateBalance() unexpected error: %v", err)
	}

	if got.Amount != 125000 {
		t.Fatalf("CalculateBalance() = %d, want %d", got.Amount, 125000)
	}
}

func TestCalculateBalanceIgnoresCancelledTransactions(t *testing.T) {
	accountID := uuid.New()
	userID := uuid.New()
	date := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	deposit, err := NewDeposit(accountID, 100000, date, userID)
	if err != nil {
		t.Fatalf("NewDeposit() unexpected error: %v", err)
	}

	withdrawal, err := NewWithdrawal(accountID, 20000, date, userID)
	if err != nil {
		t.Fatalf("NewWithdrawal() unexpected error: %v", err)
	}

	if err := withdrawal.Cancel(); err != nil {
		t.Fatalf("Cancel() unexpected error: %v", err)
	}

	got, err := CalculateBalance([]Transaction{deposit, withdrawal})
	if err != nil {
		t.Fatalf("CalculateBalance() unexpected error: %v", err)
	}

	if got.Amount != 100000 {
		t.Fatalf("CalculateBalance() = %d, want %d", got.Amount, 100000)
	}
}

func TestCalculateBalanceRejectsNegativeResult(t *testing.T) {
	accountID := uuid.New()
	userID := uuid.New()
	date := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	deposit, err := NewDeposit(accountID, 50000, date, userID)
	if err != nil {
		t.Fatalf("NewDeposit() unexpected error: %v", err)
	}

	withdrawal, err := NewWithdrawal(accountID, 70000, date, userID)
	if err != nil {
		t.Fatalf("NewWithdrawal() unexpected error: %v", err)
	}

	_, err = CalculateBalance([]Transaction{deposit, withdrawal})

	if err != ErrNegativeBalance {
		t.Fatalf("CalculateBalance() error = %v, want %v", err, ErrNegativeBalance)
	}
}

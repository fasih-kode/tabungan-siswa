package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewSavingsAccount(t *testing.T) {
	studentID := uuid.New()
	academicYearID := uuid.New()

	account, err := NewSavingsAccount(studentID, academicYearID)
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}

	if account.ID == uuid.Nil {
		t.Fatal("NewSavingsAccount() ID must not be nil")
	}

	if account.StudentID != studentID {
		t.Fatalf("NewSavingsAccount() StudentID = %v, want %v", account.StudentID, studentID)
	}

	if account.AcademicYearID != academicYearID {
		t.Fatalf("NewSavingsAccount() AcademicYearID = %v, want %v", account.AcademicYearID, academicYearID)
	}

	if account.Status != SavingsAccountOpen {
		t.Fatalf("NewSavingsAccount() Status = %v, want %v", account.Status, SavingsAccountOpen)
	}

	if account.SettledAt != nil {
		t.Fatal("NewSavingsAccount() SettledAt must be nil")
	}

	if account.CreatedAt.IsZero() {
		t.Fatal("NewSavingsAccount() CreatedAt must not be zero")
	}

	if account.UpdatedAt.IsZero() {
		t.Fatal("NewSavingsAccount() UpdatedAt must not be zero")
	}
}

func TestNewSavingsAccountRejectsNilStudentID(t *testing.T) {
	_, err := NewSavingsAccount(uuid.Nil, uuid.New())

	if err != ErrInvalidID {
		t.Fatalf("NewSavingsAccount() error = %v, want %v", err, ErrInvalidID)
	}
}

func TestNewSavingsAccountRejectsNilAcademicYearID(t *testing.T) {
	_, err := NewSavingsAccount(uuid.New(), uuid.Nil)

	if err != ErrInvalidID {
		t.Fatalf("NewSavingsAccount() error = %v, want %v", err, ErrInvalidID)
	}
}

func TestSavingsAccountSettle(t *testing.T) {
	account, err := NewSavingsAccount(uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}

	settledAt := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)

	err = account.Settle(settledAt)
	if err != nil {
		t.Fatalf("SavingsAccount.Settle() error = %v", err)
	}

	if !account.IsSettled() {
		t.Fatal("SavingsAccount should be settled")
	}

	if account.IsOpen() {
		t.Fatal("SavingsAccount should not be open after settlement")
	}

	if account.SettledAt == nil {
		t.Fatal("SavingsAccount.SettledAt must not be nil")
	}

	if !account.SettledAt.Equal(settledAt) {
		t.Fatalf("SavingsAccount.SettledAt = %v, want %v", *account.SettledAt, settledAt)
	}
}

func TestSavingsAccountCannotBeSettledTwice(t *testing.T) {
	account, err := NewSavingsAccount(uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}

	settledAt := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)

	if err := account.Settle(settledAt); err != nil {
		t.Fatalf("SavingsAccount.Settle() first call error = %v", err)
	}

	if err := account.Settle(settledAt); err != ErrAlreadySettled {
		t.Fatalf("SavingsAccount.Settle() second call error = %v, want %v", err, ErrAlreadySettled)
	}
}

func TestSavingsAccountReopen(t *testing.T) {
	account, err := NewSavingsAccount(uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}

	settledAt := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)

	if err := account.Settle(settledAt); err != nil {
		t.Fatalf("SavingsAccount.Settle() error = %v", err)
	}

	if err := account.Reopen(); err != nil {
		t.Fatalf("SavingsAccount.Reopen() error = %v", err)
	}

	if !account.IsOpen() {
		t.Fatal("SavingsAccount should be open after reopen")
	}

	if account.IsSettled() {
		t.Fatal("SavingsAccount should not be settled after reopen")
	}

	if account.SettledAt != nil {
		t.Fatal("SavingsAccount.SettledAt must be nil after reopen")
	}
}

func TestSavingsAccountCannotReopenWhenOpen(t *testing.T) {
	account, err := NewSavingsAccount(uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("NewSavingsAccount() error = %v", err)
	}

	if err := account.Reopen(); err != ErrInvalidTransition {
		t.Fatalf("SavingsAccount.Reopen() error = %v, want %v", err, ErrInvalidTransition)
	}
}

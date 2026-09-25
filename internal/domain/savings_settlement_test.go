package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewSavingsSettlement(t *testing.T) {
	accountID := uuid.New()
	userID := uuid.New()
	executedAt := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)

	settlement, err := NewSavingsSettlement(
		accountID,
		SettlementRegularYearEnd,
		100000,
		userID,
		executedAt,
		nil,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement() error = %v", err)
	}

	if settlement.ID == uuid.Nil {
		t.Fatal("settlement ID must not be nil")
	}

	if settlement.SavingsAccountID != accountID {
		t.Fatalf("SavingsAccountID = %v, want %v", settlement.SavingsAccountID, accountID)
	}

	if settlement.Type != SettlementRegularYearEnd {
		t.Fatalf("Type = %v, want %v", settlement.Type, SettlementRegularYearEnd)
	}

	if settlement.Amount != 100000 {
		t.Fatalf("Amount = %v, want 100000", settlement.Amount)
	}

	if settlement.ExecutedBy != userID {
		t.Fatalf("ExecutedBy = %v, want %v", settlement.ExecutedBy, userID)
	}

	if !settlement.ExecutedAt.Equal(executedAt) {
		t.Fatalf("ExecutedAt = %v, want %v", settlement.ExecutedAt, executedAt)
	}

	if settlement.Status != SettlementCompleted {
		t.Fatalf("Status = %v, want %v", settlement.Status, SettlementCompleted)
	}

	if settlement.ReplacesSettlementID != nil {
		t.Fatal("ReplacesSettlementID must be nil for a normal settlement")
	}

	if !settlement.IsCompleted() {
		t.Fatal("settlement should be completed")
	}

	if settlement.IsSuperseded() {
		t.Fatal("settlement should not be superseded")
	}
}

func TestNewSavingsSettlementRejectsNilSavingsAccountID(t *testing.T) {
	_, err := NewSavingsSettlement(
		uuid.Nil,
		SettlementRegularYearEnd,
		100000,
		uuid.New(),
		time.Now(),
		nil,
	)

	if err != ErrInvalidID {
		t.Fatalf(
			"NewSavingsSettlement() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}
}

func TestNewSavingsSettlementRejectsNilExecutedBy(t *testing.T) {
	_, err := NewSavingsSettlement(
		uuid.New(),
		SettlementRegularYearEnd,
		100000,
		uuid.Nil,
		time.Now(),
		nil,
	)

	if err != ErrInvalidID {
		t.Fatalf(
			"NewSavingsSettlement() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}
}

func TestNewSavingsSettlementAllowsZeroAmount(t *testing.T) {
	settlement, err := NewSavingsSettlement(
		uuid.New(),
		SettlementStudentLeaving,
		0,
		uuid.New(),
		time.Now(),
		nil,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement() error = %v", err)
	}

	if settlement.Amount != 0 {
		t.Fatalf("Amount = %v, want 0", settlement.Amount)
	}
}

func TestNewSavingsSettlementRejectsNegativeAmount(t *testing.T) {
	_, err := NewSavingsSettlement(
		uuid.New(),
		SettlementStudentLeaving,
		-1,
		uuid.New(),
		time.Now(),
		nil,
	)
	if err != ErrInvalidAmount {
		t.Fatalf("NewSavingsSettlement() error = %v, want %v", err, ErrInvalidAmount)
	}
}

func TestNewSavingsSettlementReplacement(t *testing.T) {
	accountID := uuid.New()
	userID := uuid.New()
	oldSettlementID := uuid.New()

	settlement, err := NewSavingsSettlement(
		accountID,
		SettlementRegularYearEnd,
		150000,
		userID,
		time.Now(),
		&oldSettlementID,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement() error = %v", err)
	}

	if settlement.ReplacesSettlementID == nil {
		t.Fatal("ReplacesSettlementID must not be nil")
	}

	if *settlement.ReplacesSettlementID != oldSettlementID {
		t.Fatalf(
			"ReplacesSettlementID = %v, want %v",
			*settlement.ReplacesSettlementID,
			oldSettlementID,
		)
	}

	if settlement.Status != SettlementCompleted {
		t.Fatalf("Status = %v, want %v", settlement.Status, SettlementCompleted)
	}
}

func TestSavingsSettlementSupersede(t *testing.T) {
	settlement, err := NewSavingsSettlement(
		uuid.New(),
		SettlementRegularYearEnd,
		100000,
		uuid.New(),
		time.Now(),
		nil,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement() error = %v", err)
	}

	if err := settlement.Supersede(); err != nil {
		t.Fatalf("Supersede() error = %v", err)
	}

	if settlement.Status != SettlementSuperseded {
		t.Fatalf("Status = %v, want %v", settlement.Status, SettlementSuperseded)
	}

	if !settlement.IsSuperseded() {
		t.Fatal("settlement should be superseded")
	}

	if settlement.IsCompleted() {
		t.Fatal("settlement should not be completed")
	}
}

func TestSavingsSettlementCannotSupersedeTwice(t *testing.T) {
	settlement, err := NewSavingsSettlement(
		uuid.New(),
		SettlementRegularYearEnd,
		100000,
		uuid.New(),
		time.Now(),
		nil,
	)
	if err != nil {
		t.Fatalf("NewSavingsSettlement() error = %v", err)
	}

	if err := settlement.Supersede(); err != nil {
		t.Fatalf("first Supersede() error = %v", err)
	}

	if err := settlement.Supersede(); err != ErrInvalidTransition {
		t.Fatalf("second Supersede() error = %v, want %v", err, ErrInvalidTransition)
	}
}

func TestNewSavingsSettlementRejectsNilReplacementID(t *testing.T) {
	replacementID := uuid.Nil

	_, err := NewSavingsSettlement(
		uuid.New(),
		SettlementRegularYearEnd,
		100000,
		uuid.New(),
		time.Now(),
		&replacementID,
	)
	if err != ErrInvalidID {
		t.Fatalf(
			"NewSavingsSettlement() error = %v, want %v",
			err,
			ErrInvalidID,
		)
	}
}

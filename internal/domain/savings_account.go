package domain

import (
	"time"

	"github.com/google/uuid"
)

type SavingsAccount struct {
	ID             uuid.UUID
	StudentID      uuid.UUID
	AcademicYearID uuid.UUID
	Status         SavingsAccountStatus
	SettledAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewSavingsAccount(
	studentID uuid.UUID,
	academicYearID uuid.UUID,
) (SavingsAccount, error) {
	if studentID == uuid.Nil {
		return SavingsAccount{}, ErrInvalidID
	}

	if academicYearID == uuid.Nil {
		return SavingsAccount{}, ErrInvalidID
	}

	now := time.Now()

	return SavingsAccount{
		ID:             uuid.New(),
		StudentID:      studentID,
		AcademicYearID: academicYearID,
		Status:         SavingsAccountOpen,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (a SavingsAccount) IsOpen() bool {
	return a.Status == SavingsAccountOpen
}

func (a SavingsAccount) IsSettled() bool {
	return a.Status == SavingsAccountSettled
}

func (a *SavingsAccount) Settle(at time.Time) error {
	if a.Status == SavingsAccountSettled {
		return ErrAlreadySettled
	}

	if a.Status != SavingsAccountOpen {
		return ErrInvalidTransition
	}

	a.Status = SavingsAccountSettled
	a.SettledAt = &at
	a.UpdatedAt = time.Now()

	return nil
}

func (a *SavingsAccount) Reopen() error {
	if a.Status != SavingsAccountSettled {
		return ErrInvalidTransition
	}

	a.Status = SavingsAccountOpen
	a.SettledAt = nil
	a.UpdatedAt = time.Now()

	return nil
}

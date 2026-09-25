package domain

import (
	"time"

	"github.com/google/uuid"
)

type SavingsSettlement struct {
	ID                   uuid.UUID
	SavingsAccountID     uuid.UUID
	Type                 SettlementType
	Amount               Money
	ExecutedBy           uuid.UUID
	ExecutedAt           time.Time
	Status               SettlementStatus
	ReplacesSettlementID *uuid.UUID
}

func NewSavingsSettlement(
	savingsAccountID uuid.UUID,
	settlementType SettlementType,
	amount Money,
	executedBy uuid.UUID,
	executedAt time.Time,
	replacesSettlementID *uuid.UUID,
) (SavingsSettlement, error) {
	if savingsAccountID == uuid.Nil {
		return SavingsSettlement{}, ErrInvalidID
	}

	if executedBy == uuid.Nil {
		return SavingsSettlement{}, ErrInvalidID
	}

	if !settlementType.IsValid() {
		return SavingsSettlement{}, ErrInvalidValue
	}

	if amount < 0 {
		return SavingsSettlement{}, ErrInvalidAmount
	}

	if replacesSettlementID != nil && *replacesSettlementID == uuid.Nil {
		return SavingsSettlement{}, ErrInvalidID
	}

	return SavingsSettlement{
		ID:                   uuid.New(),
		SavingsAccountID:     savingsAccountID,
		Type:                 settlementType,
		Amount:               amount,
		ExecutedBy:           executedBy,
		ExecutedAt:           executedAt,
		Status:               SettlementCompleted,
		ReplacesSettlementID: replacesSettlementID,
	}, nil
}

func (s SavingsSettlement) IsCompleted() bool {
	return s.Status == SettlementCompleted
}

func (s SavingsSettlement) IsSuperseded() bool {
	return s.Status == SettlementSuperseded
}

func (s *SavingsSettlement) Supersede() error {
	if s.Status != SettlementCompleted {
		return ErrInvalidTransition
	}

	s.Status = SettlementSuperseded

	return nil
}

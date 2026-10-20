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

type SettlementCashMovementKind string

const (
	SettlementCashHandover SettlementCashMovementKind = "HANDOVER"
	SettlementCashReturn   SettlementCashMovementKind = "RETURN"
)

// SettlementCashMovement records one physical cash movement. It is append-only:
// later corrections add a return or additional handover instead of rewriting
// the original receipt.
type SettlementCashMovement struct {
	ID                  uuid.UUID
	SavingsSettlementID uuid.UUID
	Kind                SettlementCashMovementKind
	Amount              Money
	ConfirmedBy         uuid.UUID
	ConfirmedAt         time.Time
	ReceiptReference    *string
}

func NewSettlementCashMovement(settlementID uuid.UUID, kind SettlementCashMovementKind, amount Money, actorID uuid.UUID, at time.Time, reference *string) (SettlementCashMovement, error) {
	if settlementID == uuid.Nil || actorID == uuid.Nil {
		return SettlementCashMovement{}, ErrInvalidID
	}
	if kind != SettlementCashHandover && kind != SettlementCashReturn {
		return SettlementCashMovement{}, ErrInvalidValue
	}
	if amount <= 0 || amount > MaxMoney {
		return SettlementCashMovement{}, ErrInvalidAmount
	}
	if reference != nil && len([]rune(*reference)) > 200 {
		return SettlementCashMovement{}, ErrInvalidValue
	}
	return SettlementCashMovement{ID: uuid.New(), SavingsSettlementID: settlementID, Kind: kind, Amount: amount, ConfirmedBy: actorID, ConfirmedAt: at, ReceiptReference: reference}, nil
}

type SettlementReopenRequestStatus string

const (
	SettlementReopenPending  SettlementReopenRequestStatus = "PENDING"
	SettlementReopenApproved SettlementReopenRequestStatus = "APPROVED"
	SettlementReopenRejected SettlementReopenRequestStatus = "REJECTED"
	SettlementReopenComplete SettlementReopenRequestStatus = "COMPLETED"
)

type SettlementReopenRequest struct {
	ID                  uuid.UUID
	SavingsSettlementID uuid.UUID
	SavingsAccountID    uuid.UUID
	AcademicYearID      uuid.UUID
	RequestedBy         uuid.UUID
	Reason              string
	RequestedAt         time.Time
	Status              SettlementReopenRequestStatus
	DecidedBy           *uuid.UUID
	DecidedAt           *time.Time
	DecisionReason      *string
	CompletedAt         *time.Time
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

	if amount < 0 || amount > MaxMoney {
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

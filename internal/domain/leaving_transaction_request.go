package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type LeavingTransactionRequestStatus string

const (
	LeavingTransactionPending   LeavingTransactionRequestStatus = "PENDING"
	LeavingTransactionApproved  LeavingTransactionRequestStatus = "APPROVED"
	LeavingTransactionRejected  LeavingTransactionRequestStatus = "REJECTED"
	LeavingTransactionCompleted LeavingTransactionRequestStatus = "COMPLETED"
)

type LeavingTransactionRequest struct {
	ID, SavingsAccountID, AcademicYearID uuid.UUID
	Type                                 TransactionType
	Amount                               Money
	TransactionDate                      time.Time
	Reason, EvidenceReference            string
	RequestedBy                          uuid.UUID
	RequestedAt                          time.Time
	Status                               LeavingTransactionRequestStatus
	DecidedBy                            *uuid.UUID
	DecidedAt                            *time.Time
	DecisionReason                       string
	CompletedAt                          *time.Time
}

func NewLeavingTransactionRequest(accountID, yearID uuid.UUID, txType TransactionType, amount Money, date time.Time, reason, evidence string, actorID uuid.UUID) (LeavingTransactionRequest, error) {
	reason = strings.TrimSpace(reason)
	evidence = strings.TrimSpace(evidence)
	if accountID == uuid.Nil || yearID == uuid.Nil || actorID == uuid.Nil {
		return LeavingTransactionRequest{}, ErrInvalidID
	}
	if (txType != TransactionDeposit && txType != TransactionWithdrawal) || !amount.IsPositive() || amount > MaxMoney {
		return LeavingTransactionRequest{}, ErrInvalidAmount
	}
	if date.IsZero() || len(reason) == 0 || len(reason) > 500 || len(evidence) == 0 || len(evidence) > 200 {
		return LeavingTransactionRequest{}, ErrInvalidValue
	}
	now := time.Now().UTC()
	return LeavingTransactionRequest{ID: uuid.New(), SavingsAccountID: accountID, AcademicYearID: yearID, Type: txType, Amount: amount, TransactionDate: date, Reason: reason, EvidenceReference: evidence, RequestedBy: actorID, RequestedAt: now, Status: LeavingTransactionPending}, nil
}

func (r *LeavingTransactionRequest) Decide(actorID uuid.UUID, approve bool, reason string) error {
	reason = strings.TrimSpace(reason)
	if r.Status != LeavingTransactionPending || actorID == uuid.Nil || len(reason) == 0 || len(reason) > 500 {
		return ErrInvalidTransition
	}
	now := time.Now().UTC()
	r.DecidedBy, r.DecidedAt, r.DecisionReason = &actorID, &now, reason
	if approve {
		r.Status = LeavingTransactionApproved
	} else {
		r.Status = LeavingTransactionRejected
	}
	return nil
}

func (r *LeavingTransactionRequest) Complete() error {
	if r.Status != LeavingTransactionApproved {
		return ErrInvalidTransition
	}
	now := time.Now().UTC()
	r.Status, r.CompletedAt = LeavingTransactionCompleted, &now
	return nil
}

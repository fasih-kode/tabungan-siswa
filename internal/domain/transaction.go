package domain

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID               uuid.UUID
	SavingsAccountID uuid.UUID
	Type             TransactionType
	Amount           Money
	TransactionDate  time.Time
	Status           TransactionStatus
	CreatedBy        uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewTransaction(
	savingsAccountID uuid.UUID,
	transactionType TransactionType,
	amount Money,
	transactionDate time.Time,
	createdBy uuid.UUID,
) (Transaction, error) {
	if savingsAccountID == uuid.Nil {
		return Transaction{}, ErrInvalidID
	}

	if createdBy == uuid.Nil {
		return Transaction{}, ErrInvalidID
	}

	if !transactionType.IsValid() {
		return Transaction{}, ErrInvalidValue
	}

	if !amount.IsPositive() {
		return Transaction{}, ErrInvalidAmount
	}

	now := time.Now()

	return Transaction{
		ID:               uuid.New(),
		SavingsAccountID: savingsAccountID,
		Type:             transactionType,
		Amount:           amount,
		TransactionDate:  transactionDate,
		Status:           TransactionActive,
		CreatedBy:        createdBy,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func NewDeposit(
	savingsAccountID uuid.UUID,
	amount Money,
	transactionDate time.Time,
	createdBy uuid.UUID,
) (Transaction, error) {
	return NewTransaction(
		savingsAccountID,
		TransactionDeposit,
		amount,
		transactionDate,
		createdBy,
	)
}

func NewWithdrawal(
	savingsAccountID uuid.UUID,
	amount Money,
	transactionDate time.Time,
	createdBy uuid.UUID,
) (Transaction, error) {
	return NewTransaction(
		savingsAccountID,
		TransactionWithdrawal,
		amount,
		transactionDate,
		createdBy,
	)
}

func (t Transaction) IsActive() bool {
	return t.Status == TransactionActive
}

func (t Transaction) IsCancelled() bool {
	return t.Status == TransactionCancelled
}

func (t *Transaction) Cancel() error {
	if t.Status == TransactionCancelled {
		return ErrAlreadyCancelled
	}

	if t.Status != TransactionActive {
		return ErrInvalidTransition
	}

	t.Status = TransactionCancelled
	t.UpdatedAt = time.Now()

	return nil
}

package domain

type Balance struct {
	Amount Money
}

func NewBalance(amount Money) (Balance, error) {
	if amount < 0 {
		return Balance{}, ErrNegativeBalance
	}

	return Balance{
		Amount: amount,
	}, nil
}

func CalculateBalance(transactions []Transaction) (Balance, error) {
	var amount Money

	for _, transaction := range transactions {
		if transaction.Status != TransactionActive {
			continue
		}

		switch transaction.Type {
		case TransactionDeposit:
			amount += transaction.Amount

		case TransactionWithdrawal:
			amount -= transaction.Amount

			if amount < 0 {
				return Balance{}, ErrNegativeBalance
			}
		}
	}

	return NewBalance(amount)
}

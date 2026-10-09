package domain

type Balance struct {
	Amount Money
}

func NewBalance(amount Money) (Balance, error) {
	if amount < 0 {
		return Balance{}, ErrNegativeBalance
	}
	if amount > MaxMoney {
		return Balance{}, ErrMoneyOverflow
	}
	return Balance{Amount: amount}, nil
}

func CalculateBalance(transactions []Transaction) (Balance, error) {
	var amount Money
	for _, transaction := range transactions {
		if transaction.Status != TransactionActive {
			continue
		}
		if !transaction.Amount.IsPositive() || transaction.Amount > MaxMoney {
			return Balance{}, ErrInvalidAmount
		}
		switch transaction.Type {
		case TransactionDeposit:
			var err error
			amount, err = amount.Add(transaction.Amount)
			if err != nil {
				return Balance{}, err
			}
		case TransactionWithdrawal:
			var err error
			amount, err = amount.Subtract(transaction.Amount)
			if err != nil {
				return Balance{}, err
			}
		default:
			return Balance{}, ErrInvalidValue
		}
	}
	return NewBalance(amount)
}

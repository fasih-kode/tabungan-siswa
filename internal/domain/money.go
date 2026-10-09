package domain

// MaxMoney matches the largest whole-number amount representable by NUMERIC(15,2).
const MaxMoney Money = 9_999_999_999_999

type Money int64

func (m Money) IsNegative() bool { return m < 0 }
func (m Money) IsZero() bool     { return m == 0 }
func (m Money) IsPositive() bool { return m > 0 }

func (m Money) Add(other Money) (Money, error) {
	if m > MaxMoney || m < -MaxMoney || other > MaxMoney || other < -MaxMoney {
		return 0, ErrMoneyOverflow
	}
	if other > 0 && m > MaxMoney-other || other < 0 && m < -MaxMoney-other {
		return 0, ErrMoneyOverflow
	}
	result := m + other
	if result > MaxMoney || result < -MaxMoney {
		return 0, ErrMoneyOverflow
	}
	return result, nil
}

func (m Money) Subtract(other Money) (Money, error) {
	if m > MaxMoney || m < -MaxMoney || other > MaxMoney || other < -MaxMoney {
		return 0, ErrMoneyOverflow
	}
	if other < 0 && m > MaxMoney+other || other > 0 && m < -MaxMoney+other {
		return 0, ErrMoneyOverflow
	}
	result := m - other
	if result > MaxMoney || result < -MaxMoney {
		return 0, ErrMoneyOverflow
	}
	if result < 0 {
		return 0, ErrNegativeBalance
	}
	return result, nil
}

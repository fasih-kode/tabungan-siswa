package domain

type Money int64

func (m Money) IsNegative() bool {
	return m < 0
}

func (m Money) IsZero() bool {
	return m == 0
}

func (m Money) IsPositive() bool {
	return m > 0
}

func (m Money) Add(other Money) Money {
	return m + other
}

func (m Money) Subtract(other Money) (Money, error) {
	result := m - other
	if result < 0 {
		return 0, ErrNegativeBalance
	}

	return result, nil
}

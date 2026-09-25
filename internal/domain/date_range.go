package domain

import "time"

type DateRange struct {
	From time.Time
	To   *time.Time
}

func NewDateRange(from time.Time, to *time.Time) (DateRange, error) {
	if to != nil && !from.Before(*to) {
		return DateRange{}, ErrInvalidDateRange
	}

	return DateRange{
		From: from,
		To:   to,
	}, nil
}

func (r DateRange) Contains(date time.Time) bool {
	if date.Before(r.From) {
		return false
	}

	if r.To == nil {
		return true
	}

	return date.Before(*r.To)
}

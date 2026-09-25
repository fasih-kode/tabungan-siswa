package domain

import (
	"testing"
	"time"
)

func TestNewDateRange(t *testing.T) {
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)

	got, err := NewDateRange(from, &to)
	if err != nil {
		t.Fatalf("NewDateRange() unexpected error: %v", err)
	}

	if !got.From.Equal(from) {
		t.Fatalf("From = %v, want %v", got.From, from)
	}

	if got.To == nil || !got.To.Equal(to) {
		t.Fatalf("To = %v, want %v", got.To, to)
	}
}

func TestNewDateRangeAllowsOpenEndedRange(t *testing.T) {
	from := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)

	got, err := NewDateRange(from, nil)
	if err != nil {
		t.Fatalf("NewDateRange() unexpected error: %v", err)
	}

	if !got.From.Equal(from) {
		t.Fatalf("From = %v, want %v", got.From, from)
	}

	if got.To != nil {
		t.Fatalf("To = %v, want nil", got.To)
	}
}

func TestNewDateRangeRejectsInvalidRange(t *testing.T) {
	from := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	_, err := NewDateRange(from, &to)

	if err != ErrInvalidDateRange {
		t.Fatalf("NewDateRange() error = %v, want %v", err, ErrInvalidDateRange)
	}
}

func TestNewDateRangeRejectsSameDate(t *testing.T) {
	date := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)

	_, err := NewDateRange(date, &date)

	if err != ErrInvalidDateRange {
		t.Fatalf("NewDateRange() error = %v, want %v", err, ErrInvalidDateRange)
	}
}

func TestDateRangeContains(t *testing.T) {
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)

	r, err := NewDateRange(from, &to)
	if err != nil {
		t.Fatalf("NewDateRange() unexpected error: %v", err)
	}

	tests := []struct {
		name string
		date time.Time
		want bool
	}{
		{
			name: "before",
			date: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "from inclusive",
			date: from,
			want: true,
		},
		{
			name: "inside",
			date: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "to exclusive",
			date: to,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.Contains(tt.date); got != tt.want {
				t.Fatalf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

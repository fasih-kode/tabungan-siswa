package domain

import "testing"

func TestMoneyIsPositive(t *testing.T) {
	tests := []struct {
		name  string
		money Money
		want  bool
	}{
		{name: "positive", money: 50000, want: true},
		{name: "zero", money: 0, want: false},
		{name: "negative", money: -1, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.money.IsPositive(); got != tt.want {
				t.Fatalf("IsPositive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMoneyAdd(t *testing.T) {
	got := Money(10000).Add(5000)

	if got != 15000 {
		t.Fatalf("Add() = %d, want %d", got, 15000)
	}
}

func TestMoneySubtract(t *testing.T) {
	got, err := Money(10000).Subtract(3000)
	if err != nil {
		t.Fatalf("Subtract() unexpected error: %v", err)
	}

	if got != 7000 {
		t.Fatalf("Subtract() = %d, want %d", got, 7000)
	}
}

func TestMoneySubtractBelowZero(t *testing.T) {
	_, err := Money(3000).Subtract(5000)

	if err != ErrNegativeBalance {
		t.Fatalf("Subtract() error = %v, want %v", err, ErrNegativeBalance)
	}
}

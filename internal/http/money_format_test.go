package http

import (
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
)

func TestFormatRupiahUsesIndonesianThousandsSeparator(t *testing.T) {
	tests := []struct {
		amount domain.Money
		want   string
	}{
		{amount: 0, want: "Rp. 0"},
		{amount: 10_000, want: "Rp. 10.000"},
		{amount: 1_250_000, want: "Rp. 1.250.000"},
	}
	for _, tt := range tests {
		if got := formatRupiah(tt.amount); got != tt.want {
			t.Errorf("formatRupiah(%d) = %q, want %q", tt.amount, got, tt.want)
		}
	}
}

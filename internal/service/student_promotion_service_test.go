package service

import (
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/domain"
)

func TestValidatePromotionYears(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)
	from := domain.AcademicYear{Status: domain.AcademicYearClosing, EndDate: start}
	to := domain.AcademicYear{Status: domain.AcademicYearOpen, StartDate: start, EndDate: end}
	if err := validatePromotionYears(from, to); err != nil {
		t.Fatalf("adjacent closing/open years rejected: %v", err)
	}

	tests := []struct {
		name     string
		from, to domain.AcademicYear
	}{
		{name: "source still open", from: domain.AcademicYear{Status: domain.AcademicYearOpen, EndDate: start}, to: to},
		{name: "target not open", from: from, to: domain.AcademicYear{Status: domain.AcademicYearClosed, StartDate: start, EndDate: end}},
		{name: "overlapping years", from: domain.AcademicYear{Status: domain.AcademicYearClosed, EndDate: start}, to: domain.AcademicYear{Status: domain.AcademicYearOpen, StartDate: start.Add(-time.Hour), EndDate: end}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validatePromotionYears(tt.from, tt.to); err == nil {
				t.Fatal("expected invalid year transition")
			}
		})
	}
}

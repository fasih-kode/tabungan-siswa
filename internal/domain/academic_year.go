package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type AcademicYear struct {
	ID        uuid.UUID
	Name      string
	StartDate time.Time
	EndDate   time.Time
	Status    AcademicYearStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewAcademicYear(
	name string,
	startDate time.Time,
	endDate time.Time,
) (AcademicYear, error) {
	name = strings.TrimSpace(name)

	if name == "" {
		return AcademicYear{}, ErrEmptyName
	}

	if !startDate.Before(endDate) {
		return AcademicYear{}, ErrInvalidDateRange
	}

	now := time.Now()

	return AcademicYear{
		ID:        uuid.New(),
		Name:      name,
		StartDate: startDate,
		EndDate:   endDate,
		Status:    AcademicYearOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (y AcademicYear) IsOpen() bool {
	return y.Status == AcademicYearOpen
}

func (y AcademicYear) IsClosing() bool {
	return y.Status == AcademicYearClosing
}

func (y AcademicYear) IsClosed() bool {
	return y.Status == AcademicYearClosed
}

func (y *AcademicYear) StartClosing() error {
	if y.Status != AcademicYearOpen {
		return ErrInvalidTransition
	}

	y.Status = AcademicYearClosing
	y.UpdatedAt = time.Now()

	return nil
}

func (y *AcademicYear) Close() error {
	if y.Status != AcademicYearClosing {
		return ErrInvalidTransition
	}

	y.Status = AcademicYearClosed
	y.UpdatedAt = time.Now()

	return nil
}

func (y *AcademicYear) Reopen() error {
	if y.Status != AcademicYearClosed {
		return ErrInvalidTransition
	}

	y.Status = AcademicYearOpen
	y.UpdatedAt = time.Now()

	return nil
}

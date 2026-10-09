package domain

import (
	"time"

	"github.com/google/uuid"
)

type StudentClassHistory struct {
	ID             uuid.UUID
	StudentID      uuid.UUID
	AcademicYearID uuid.UUID
	ClassID        uuid.UUID
	Period         DateRange
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewStudentClassHistory(
	studentID uuid.UUID,
	academicYearID uuid.UUID,
	classID uuid.UUID,
	period DateRange,
) (StudentClassHistory, error) {
	if studentID == uuid.Nil {
		return StudentClassHistory{}, ErrInvalidID
	}

	if academicYearID == uuid.Nil {
		return StudentClassHistory{}, ErrInvalidID
	}

	if classID == uuid.Nil {
		return StudentClassHistory{}, ErrInvalidID
	}

	now := time.Now()

	return StudentClassHistory{
		ID:             uuid.New(),
		StudentID:      studentID,
		AcademicYearID: academicYearID,
		ClassID:        classID,
		Period:         period,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (h StudentClassHistory) Contains(date time.Time) bool {
	return h.Period.Contains(date)
}

// ValidateWithinAcademicYear enforces the half-open academic-year boundary.
func (h StudentClassHistory) ValidateWithinAcademicYear(startDate, endDate time.Time) error {
	if !startDate.Before(endDate) || h.Period.From.Before(startDate) || !h.Period.From.Before(endDate) {
		return ErrInvalidDateRange
	}
	if h.Period.To == nil || !h.Period.To.After(h.Period.From) || h.Period.To.After(endDate) {
		return ErrInvalidDateRange
	}
	return nil
}

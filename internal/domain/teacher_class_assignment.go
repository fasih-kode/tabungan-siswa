package domain

import (
	"time"

	"github.com/google/uuid"
)

type TeacherClassAssignment struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	AcademicYearID uuid.UUID
	ClassID        uuid.UUID
	Period         DateRange
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewTeacherClassAssignment(
	userID uuid.UUID,
	academicYearID uuid.UUID,
	classID uuid.UUID,
	period DateRange,
) (TeacherClassAssignment, error) {
	if userID == uuid.Nil {
		return TeacherClassAssignment{}, ErrInvalidID
	}

	if academicYearID == uuid.Nil {
		return TeacherClassAssignment{}, ErrInvalidID
	}

	if classID == uuid.Nil {
		return TeacherClassAssignment{}, ErrInvalidID
	}

	now := time.Now()

	return TeacherClassAssignment{
		ID:             uuid.New(),
		UserID:         userID,
		AcademicYearID: academicYearID,
		ClassID:        classID,
		Period:         period,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (a TeacherClassAssignment) Contains(date time.Time) bool {
	return a.Period.Contains(date)
}

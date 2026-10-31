package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Student struct {
	ID          uuid.UUID
	UserID      *uuid.UUID
	NIS         *string
	NISN        *string
	Name        string
	Status      StudentStatus
	LeavingDate *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewStudent(
	name string,
	userID *uuid.UUID,
	nis *string,
	nisn *string,
) (Student, error) {
	if strings.TrimSpace(name) == "" {
		return Student{}, ErrEmptyName
	}
	if nis == nil || strings.TrimSpace(*nis) == "" {
		return Student{}, ErrInvalidValue
	}
	if nisn == nil || strings.TrimSpace(*nisn) == "" {
		return Student{}, ErrInvalidValue
	}

	now := time.Now()

	return Student{
		ID:        uuid.New(),
		UserID:    userID,
		NIS:       nis,
		NISN:      nisn,
		Name:      name,
		Status:    StudentActive,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s Student) IsActive() bool {
	return s.Status == StudentActive
}

func (s Student) IsLeft() bool {
	return s.Status == StudentLeft
}

func (s Student) IsGraduated() bool {
	return s.Status == StudentGraduated
}

func (s *Student) Leave() error {
	return s.LeaveOn(time.Time{})
}

func (s *Student) LeaveOn(leavingDate time.Time) error {
	if s.Status != StudentActive {
		return ErrInvalidTransition
	}

	s.Status = StudentLeft
	if !leavingDate.IsZero() {
		date := time.Date(leavingDate.Year(), leavingDate.Month(), leavingDate.Day(), 0, 0, 0, 0, time.UTC)
		s.LeavingDate = &date
	}
	s.UpdatedAt = time.Now()

	return nil
}

func (s *Student) Reactivate() error {
	if s.Status != StudentLeft {
		return ErrInvalidTransition
	}

	s.Status = StudentActive
	s.UpdatedAt = time.Now()
	return nil
}

func (s *Student) Graduate() error {
	if s.Status != StudentActive {
		return ErrInvalidTransition
	}
	s.Status = StudentGraduated
	s.UpdatedAt = time.Now()
	return nil
}

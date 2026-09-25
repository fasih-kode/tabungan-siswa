package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Student struct {
	ID        uuid.UUID
	UserID    *uuid.UUID
	NIS       *string
	NISN      *string
	Name      string
	Status    StudentStatus
	CreatedAt time.Time
	UpdatedAt time.Time
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

func (s *Student) Leave() error {
	if s.Status != StudentActive {
		return ErrInvalidTransition
	}

	s.Status = StudentLeft
	s.UpdatedAt = time.Now()

	return nil
}

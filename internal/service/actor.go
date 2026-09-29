package service

import (
	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/google/uuid"
)

type Actor struct {
	UserID uuid.UUID
	Role   domain.UserRole
}

func (a Actor) Validate() error {
	if a.UserID == uuid.Nil {
		return ErrInvalidActor
	}

	if !a.Role.IsValid() {
		return ErrInvalidActor
	}

	return nil
}

package service

import (
	"github.com/fasih/tabungan-siswa/internal/domain"
	"github.com/google/uuid"
)

// RequireRole validates the actor and requires the actor to have one of the
// allowed roles.
func RequireRole(actor Actor, roles ...domain.UserRole) error {
	if err := actor.Validate(); err != nil {
		return err
	}

	for _, role := range roles {
		if actor.Role == role {
			return nil
		}
	}

	return ErrForbidden
}

// RequireSelf validates the actor and requires the target user to be the
// actor's own user ID.
func RequireSelf(actor Actor, targetUserID uuid.UUID) error {
	if err := actor.Validate(); err != nil {
		return err
	}

	if targetUserID == uuid.Nil {
		return ErrScopeViolation
	}

	if actor.UserID != targetUserID {
		return ErrScopeViolation
	}

	return nil
}

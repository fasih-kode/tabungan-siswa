package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
	Role         UserRole
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewUser(
	username string,
	passwordHash string,
	role UserRole,
) (User, error) {
	if strings.TrimSpace(username) == "" {
		return User{}, ErrEmptyName
	}

	if strings.TrimSpace(passwordHash) == "" {
		return User{}, ErrInvalidAmount
	}

	if !role.IsValid() {
		return User{}, ErrInvalidValue
	}

	now := time.Now()

	return User{
		ID:           uuid.New(),
		Username:     username,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

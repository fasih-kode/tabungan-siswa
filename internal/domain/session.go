package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

func NewSession(
	userID uuid.UUID,
	tokenHash string,
	expiresAt time.Time,
) (Session, error) {
	if userID == uuid.Nil {
		return Session{}, ErrInvalidID
	}

	if strings.TrimSpace(tokenHash) == "" {
		return Session{}, ErrInvalidSessionTokenHash
	}

	if !expiresAt.After(time.Now()) {
		return Session{}, ErrInvalidSessionExpiration
	}

	return Session{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}, nil
}

func (s Session) IsRevoked() bool {
	return s.RevokedAt != nil
}

func (s Session) IsExpired(at time.Time) bool {
	return !at.Before(s.ExpiresAt)
}

func (s Session) IsActive(at time.Time) bool {
	return !s.IsRevoked() && !s.IsExpired(at)
}

func (s *Session) Revoke(at time.Time) error {
	if s.RevokedAt != nil {
		return ErrSessionAlreadyRevoked
	}

	s.RevokedAt = &at

	return nil
}

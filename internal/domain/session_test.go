package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewSession(t *testing.T) {
	userID := uuid.New()
	expiresAt := time.Now().Add(time.Hour)

	session, err := NewSession(userID, "token-hash", expiresAt)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	if session.ID == uuid.Nil {
		t.Fatal("NewSession() ID must not be nil")
	}

	if session.UserID != userID {
		t.Fatalf("NewSession() UserID = %v, want %v", session.UserID, userID)
	}

	if session.TokenHash != "token-hash" {
		t.Fatalf("NewSession() TokenHash = %q, want %q", session.TokenHash, "token-hash")
	}

	if !session.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("NewSession() ExpiresAt = %v, want %v", session.ExpiresAt, expiresAt)
	}

	if session.RevokedAt != nil {
		t.Fatal("NewSession() RevokedAt must be nil")
	}

	if session.CreatedAt.IsZero() {
		t.Fatal("NewSession() CreatedAt must not be zero")
	}
}

func TestNewSessionRejectsNilUserID(t *testing.T) {
	_, err := NewSession(uuid.Nil, "token-hash", time.Now().Add(time.Hour))

	if err != ErrInvalidID {
		t.Fatalf("NewSession() error = %v, want %v", err, ErrInvalidID)
	}
}

func TestNewSessionRejectsEmptyTokenHash(t *testing.T) {
	_, err := NewSession(uuid.New(), "   ", time.Now().Add(time.Hour))

	if err != ErrInvalidSessionTokenHash {
		t.Fatalf(
			"NewSession() error = %v, want %v",
			err,
			ErrInvalidSessionTokenHash,
		)
	}
}

func TestNewSessionRejectsExpiredExpiration(t *testing.T) {
	_, err := NewSession(uuid.New(), "token-hash", time.Now().Add(-time.Second))

	if err != ErrInvalidSessionExpiration {
		t.Fatalf(
			"NewSession() error = %v, want %v",
			err,
			ErrInvalidSessionExpiration,
		)
	}
}

func TestNewSessionRejectsCurrentExpiration(t *testing.T) {
	expiresAt := time.Now()

	_, err := NewSession(uuid.New(), "token-hash", expiresAt)

	if err != ErrInvalidSessionExpiration {
		t.Fatalf(
			"NewSession() error = %v, want %v",
			err,
			ErrInvalidSessionExpiration,
		)
	}
}

func TestSessionIsRevoked(t *testing.T) {
	session, err := NewSession(uuid.New(), "token-hash", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	if session.IsRevoked() {
		t.Fatal("Session.IsRevoked() = true, want false")
	}

	revokedAt := time.Now()
	if err := session.Revoke(revokedAt); err != nil {
		t.Fatalf("Session.Revoke() error = %v", err)
	}

	if !session.IsRevoked() {
		t.Fatal("Session.IsRevoked() = false, want true")
	}

	if session.RevokedAt == nil || !session.RevokedAt.Equal(revokedAt) {
		t.Fatalf("Session.RevokedAt = %v, want %v", session.RevokedAt, revokedAt)
	}
}

func TestSessionIsExpired(t *testing.T) {
	expiresAt := time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC)
	session, err := NewSession(uuid.New(), "token-hash", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	session.ExpiresAt = expiresAt

	before := expiresAt.Add(-time.Nanosecond)
	at := expiresAt
	after := expiresAt.Add(time.Nanosecond)

	if session.IsExpired(before) {
		t.Fatal("Session.IsExpired(before expiration) = true, want false")
	}

	if !session.IsExpired(at) {
		t.Fatal("Session.IsExpired(at expiration) = false, want true")
	}

	if !session.IsExpired(after) {
		t.Fatal("Session.IsExpired(after expiration) = false, want true")
	}
}

func TestSessionIsActive(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour)
	session, err := NewSession(uuid.New(), "token-hash", expiresAt)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	if !session.IsActive(time.Now()) {
		t.Fatal("Session.IsActive() = false, want true")
	}

	if err := session.Revoke(time.Now()); err != nil {
		t.Fatalf("Session.Revoke() error = %v", err)
	}

	if session.IsActive(time.Now()) {
		t.Fatal("Session.IsActive() after revoke = true, want false")
	}
}

func TestSessionIsActiveWhenExpired(t *testing.T) {
	session, err := NewSession(uuid.New(), "token-hash", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	expiredAt := session.ExpiresAt
	if session.IsActive(expiredAt) {
		t.Fatal("Session.IsActive() at expiration = true, want false")
	}
}

func TestSessionCannotRevokeTwice(t *testing.T) {
	session, err := NewSession(uuid.New(), "token-hash", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	firstRevokedAt := time.Now()
	if err := session.Revoke(firstRevokedAt); err != nil {
		t.Fatalf("Session.Revoke() first call error = %v", err)
	}

	secondRevokedAt := firstRevokedAt.Add(time.Minute)
	if err := session.Revoke(secondRevokedAt); err != ErrSessionAlreadyRevoked {
		t.Fatalf(
			"Session.Revoke() second call error = %v, want %v",
			err,
			ErrSessionAlreadyRevoked,
		)
	}

	if session.RevokedAt == nil || !session.RevokedAt.Equal(firstRevokedAt) {
		t.Fatalf("Session.RevokedAt = %v, want %v", session.RevokedAt, firstRevokedAt)
	}
}

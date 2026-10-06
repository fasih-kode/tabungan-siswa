package security

import (
	"errors"
	"testing"
	"time"
)

func TestNewSessionExpirationPolicy(t *testing.T) {
	lifetime := 24 * time.Hour

	policy, err := NewSessionExpirationPolicy(lifetime)
	if err != nil {
		t.Fatalf("NewSessionExpirationPolicy() error = %v, want nil", err)
	}

	now := time.Date(2026, 10, 6, 10, 0, 0, 123456789, time.FixedZone("WIB", 7*60*60))
	got := policy.ExpiresAt(now)
	want := now.Add(lifetime)

	if !got.Equal(want) {
		t.Fatalf("ExpiresAt() = %v, want %v", got, want)
	}
}

func TestNewSessionExpirationPolicyRejectsZeroLifetime(t *testing.T) {
	_, err := NewSessionExpirationPolicy(0)

	if !errors.Is(err, ErrInvalidSessionLifetime) {
		t.Fatalf(
			"NewSessionExpirationPolicy() error = %v, want %v",
			err,
			ErrInvalidSessionLifetime,
		)
	}
}

func TestNewSessionExpirationPolicyRejectsNegativeLifetime(t *testing.T) {
	_, err := NewSessionExpirationPolicy(-time.Second)

	if !errors.Is(err, ErrInvalidSessionLifetime) {
		t.Fatalf(
			"NewSessionExpirationPolicy() error = %v, want %v",
			err,
			ErrInvalidSessionLifetime,
		)
	}
}

func TestSessionExpirationPolicyIsIndependentFromClock(t *testing.T) {
	policy, err := NewSessionExpirationPolicy(30 * time.Minute)
	if err != nil {
		t.Fatalf("NewSessionExpirationPolicy() error = %v", err)
	}

	first := time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC)
	second := first.Add(3 * time.Hour)

	if got, want := policy.ExpiresAt(first), first.Add(30*time.Minute); !got.Equal(want) {
		t.Fatalf("ExpiresAt(first) = %v, want %v", got, want)
	}

	if got, want := policy.ExpiresAt(second), second.Add(30*time.Minute); !got.Equal(want) {
		t.Fatalf("ExpiresAt(second) = %v, want %v", got, want)
	}
}

package security_test

import (
	"testing"
	"time"

	"github.com/fasih/tabungan-siswa/internal/security"
)

func TestNewSecuritySet(t *testing.T) {
	set, err := security.NewSecuritySet(24 * time.Hour)
	if err != nil {
		t.Fatalf("NewSecuritySet() error = %v", err)
	}

	if set.PasswordHasher == nil {
		t.Fatal("SecuritySet.PasswordHasher is nil")
	}

	if got := set.SessionExpiration.ExpiresAt(time.Unix(0, 0)); !got.Equal(
		time.Unix(0, 0).Add(24 * time.Hour),
	) {
		t.Fatalf(
			"SessionExpiration.ExpiresAt() = %v, want %v",
			got,
			time.Unix(0, 0).Add(24*time.Hour),
		)
	}
}

func TestNewSecuritySetRejectsInvalidSessionLifetime(t *testing.T) {
	_, err := security.NewSecuritySet(0)
	if err == nil {
		t.Fatal("NewSecuritySet() error = nil, want error")
	}
}

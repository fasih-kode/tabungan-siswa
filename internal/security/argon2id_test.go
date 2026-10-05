package security

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/fasih/tabungan-siswa/internal/domain"
)

func newTestHasher(t *testing.T) *Argon2idHasher {
	t.Helper()

	hasher, err := NewDefaultArgon2idHasher()
	if err != nil {
		t.Fatalf("NewDefaultArgon2idHasher() error = %v", err)
	}

	return hasher
}

func TestNewArgon2idHasherRejectsInvalidConfig(t *testing.T) {
	base := DefaultArgon2idConfig()

	tests := []struct {
		name   string
		config Argon2idConfig
	}{
		{
			name: "zero memory",
			config: func() Argon2idConfig {
				c := base
				c.Memory = 0
				return c
			}(),
		},
		{
			name: "zero iterations",
			config: func() Argon2idConfig {
				c := base
				c.Iterations = 0
				return c
			}(),
		},
		{
			name: "zero parallelism",
			config: func() Argon2idConfig {
				c := base
				c.Parallelism = 0
				return c
			}(),
		},
		{
			name: "short salt",
			config: func() Argon2idConfig {
				c := base
				c.SaltLength = 15
				return c
			}(),
		},
		{
			name: "short key",
			config: func() Argon2idConfig {
				c := base
				c.KeyLength = 15
				return c
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewArgon2idHasher(tt.config); err == nil {
				t.Fatal("NewArgon2idHasher() error = nil, want error")
			}
		})
	}
}

func TestNewArgon2idHasherRejectsParametersAboveMaximum(t *testing.T) {
	base := DefaultArgon2idConfig()

	tests := []struct {
		name   string
		config Argon2idConfig
	}{
		{
			name: "memory",
			config: func() Argon2idConfig {
				c := base
				c.Memory = maxArgon2idMemory + 1
				return c
			}(),
		},
		{
			name: "iterations",
			config: func() Argon2idConfig {
				c := base
				c.Iterations = maxArgon2idIterations + 1
				return c
			}(),
		},
		{
			name: "parallelism",
			config: func() Argon2idConfig {
				c := base
				c.Parallelism = maxArgon2idParallelism + 1
				return c
			}(),
		},
		{
			name: "salt length",
			config: func() Argon2idConfig {
				c := base
				c.SaltLength = maxArgon2idSaltLength + 1
				return c
			}(),
		},
		{
			name: "key length",
			config: func() Argon2idConfig {
				c := base
				c.KeyLength = maxArgon2idKeyLength + 1
				return c
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewArgon2idHasher(tt.config); err == nil {
				t.Fatal("NewArgon2idHasher() error = nil, want error")
			}
		})
	}
}

func TestArgon2idHasherHashProducesValidEncodedHash(t *testing.T) {
	hasher := newTestHasher(t)

	hash, err := hasher.Hash("password-aman-123")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Fatalf("Hash() = %q, want Argon2id PHC-style prefix", hash)
	}

	if err := hasher.Compare(hash, "password-aman-123"); err != nil {
		t.Fatalf("Compare() error = %v, want nil", err)
	}
}

func TestArgon2idHasherUsesUniqueSalt(t *testing.T) {
	hasher := newTestHasher(t)

	first, err := hasher.Hash("same-password")
	if err != nil {
		t.Fatalf("first Hash() error = %v", err)
	}

	second, err := hasher.Hash("same-password")
	if err != nil {
		t.Fatalf("second Hash() error = %v", err)
	}

	if first == second {
		t.Fatal("Hash() returned identical hashes for the same password; salt is not unique")
	}
}

func TestArgon2idHasherCompareRejectsWrongPassword(t *testing.T) {
	hasher := newTestHasher(t)

	hash, err := hasher.Hash("correct-password")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	err = hasher.Compare(hash, "wrong-password")
	if !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("Compare() error = %v, want ErrPasswordMismatch", err)
	}
}

func TestArgon2idHasherCompareRejectsMalformedHash(t *testing.T) {
	hasher := newTestHasher(t)

	err := hasher.Compare("not-a-password-hash", "password")
	if !errors.Is(err, domain.ErrInvalidPasswordHash) {
		t.Fatalf("Compare() error = %v, want domain.ErrInvalidPasswordHash", err)
	}
}

func TestArgon2idHasherRejectsEmptyPassword(t *testing.T) {
	hasher := newTestHasher(t)

	if _, err := hasher.Hash(""); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("Hash() error = %v, want ErrInvalidPassword", err)
	}

	if err := hasher.Compare("$argon2id$v=19$m=19456,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", ""); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("Compare() error = %v, want ErrInvalidPassword", err)
	}
}

func TestArgon2idHasherRejectsPasswordLongerThanMaximum(t *testing.T) {
	hasher := newTestHasher(t)

	password := strings.Repeat("a", MaxPasswordBytes+1)

	if _, err := hasher.Hash(password); !errors.Is(err, ErrPasswordTooLong) {
		t.Fatalf("Hash() error = %v, want ErrPasswordTooLong", err)
	}

	if err := hasher.Compare("not-used", password); !errors.Is(err, ErrPasswordTooLong) {
		t.Fatalf("Compare() error = %v, want ErrPasswordTooLong", err)
	}
}

func TestArgon2idHasherSupportsUnicodeAndWhitespace(t *testing.T) {
	hasher := newTestHasher(t)

	password := " كلمة سر aman \t"
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if err := hasher.Compare(hash, password); err != nil {
		t.Fatalf("Compare() error = %v, want nil", err)
	}
}

func TestArgon2idHasherRejectsTamperedHash(t *testing.T) {
	hasher := newTestHasher(t)

	hash, err := hasher.Hash("tamper-test")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Fatalf("Hash() = %q, want six PHC-style components", hash)
	}

	decoded, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(decoded) == 0 {
		t.Fatalf("decode derived key: err = %v, len = %d", err, len(decoded))
	}

	decoded[0] ^= 0x01
	parts[5] = base64.RawStdEncoding.EncodeToString(decoded)
	tampered := strings.Join(parts, "$")

	if err := hasher.Compare(tampered, "tamper-test"); !errors.Is(err, ErrPasswordMismatch) &&
		!errors.Is(err, domain.ErrInvalidPasswordHash) {
		t.Fatalf("Compare() error = %v, want mismatch or invalid hash", err)
	}
}

package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewUser(t *testing.T) {
	user, err := NewUser(
		"admin",
		"$argon2id$v=19$m=65536,t=3,p=2$example",
		RoleAdmin,
	)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}

	if user.ID == uuid.Nil {
		t.Fatal("user ID must not be nil")
	}

	if user.Username != "admin" {
		t.Fatalf("Username = %q, want %q", user.Username, "admin")
	}

	if user.PasswordHash == "" {
		t.Fatal("PasswordHash must not be empty")
	}

	if user.Role != RoleAdmin {
		t.Fatalf("Role = %v, want %v", user.Role, RoleAdmin)
	}
}

func TestNewUserAllowsAllRoles(t *testing.T) {
	roles := []UserRole{
		RoleAdmin,
		RoleWaliKelas,
		RoleSiswa,
	}

	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			_, err := NewUser(
				"user",
				"password-hash",
				role,
			)
			if err != nil {
				t.Fatalf("NewUser() error = %v", err)
			}
		})
	}
}

func TestNewUserRejectsEmptyUsername(t *testing.T) {
	_, err := NewUser(
		"",
		"password-hash",
		RoleAdmin,
	)
	if err != ErrEmptyName {
		t.Fatalf(
			"NewUser() error = %v, want %v",
			err,
			ErrEmptyName,
		)
	}
}

func TestNewUserRejectsWhitespaceUsername(t *testing.T) {
	_, err := NewUser(
		"   ",
		"password-hash",
		RoleAdmin,
	)
	if err != ErrEmptyName {
		t.Fatalf(
			"NewUser() error = %v, want %v",
			err,
			ErrEmptyName,
		)
	}
}

func TestNewUserRejectsEmptyPasswordHash(t *testing.T) {
	_, err := NewUser(
		"admin",
		"",
		RoleAdmin,
	)
	if err != ErrInvalidAmount {
		t.Fatalf(
			"NewUser() error = %v, want %v",
			err,
			ErrInvalidAmount,
		)
	}
}

func TestNewUserRejectsWhitespacePasswordHash(t *testing.T) {
	_, err := NewUser(
		"admin",
		"   ",
		RoleAdmin,
	)
	if err != ErrInvalidAmount {
		t.Fatalf(
			"NewUser() error = %v, want %v",
			err,
			ErrInvalidAmount,
		)
	}
}

func TestNewUserRejectsInvalidRole(t *testing.T) {
	invalidRole := UserRole("INVALID")

	_, err := NewUser(
		"admin",
		"password-hash",
		invalidRole,
	)
	if err != ErrInvalidValue {
		t.Fatalf(
			"NewUser() error = %v, want %v",
			err,
			ErrInvalidValue,
		)
	}
}

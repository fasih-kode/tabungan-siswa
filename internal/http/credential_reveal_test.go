package http

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestTemporaryPasswordsAreRandomAndURLSafe(t *testing.T) {
	first, err := generateTemporaryPassword()
	if err != nil {
		t.Fatal(err)
	}
	second, err := generateTemporaryPassword()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) < 12 || first == second || strings.ContainsAny(first, "+/=") {
		t.Fatalf("temporary passwords are not unique, strong, and URL-safe: %q", first)
	}
}

func TestCredentialRevealIsBoundToAdminAndConsumedOnce(t *testing.T) {
	store := newCredentialRevealStore()
	adminID := uuid.New()
	token, err := store.Put(adminID, "wali-test", "temporary-password", true)
	if err != nil {
		t.Fatal(err)
	}
	if reveal, ok := store.Take(uuid.New(), token); ok || reveal.Password != "" {
		t.Fatal("credential was available to a different admin")
	}
	reveal, ok := store.Take(adminID, token)
	if !ok || reveal.Username != "wali-test" || reveal.Password != "temporary-password" || !reveal.PasswordChangeRequired {
		t.Fatalf("credential reveal = %+v, ok=%v", reveal, ok)
	}
	if _, ok := store.Take(adminID, token); ok {
		t.Fatal("credential was available more than once")
	}
}

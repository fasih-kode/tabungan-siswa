package security

import "testing"

func TestHashSessionToken(t *testing.T) {
	first := HashSessionToken("opaque-session-token")
	second := HashSessionToken("opaque-session-token")

	if first != second {
		t.Fatalf("HashSessionToken() hashes differ: %q != %q", first, second)
	}

	if first == "opaque-session-token" {
		t.Fatal("HashSessionToken() returned plaintext token")
	}
}

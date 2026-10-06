package security

import "testing"

func TestGenerateSessionToken(t *testing.T) {
	first, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken() error = %v", err)
	}

	second, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken() second error = %v", err)
	}

	if first == second {
		t.Fatal("GenerateSessionToken() returned duplicate tokens")
	}

	if len(first) < 40 {
		t.Fatalf("GenerateSessionToken() length = %d, want at least 40", len(first))
	}

	for _, token := range []string{first, second} {
		if token == "" {
			t.Fatal("GenerateSessionToken() returned empty token")
		}
		for _, char := range token {
			if char == '+' || char == '/' || char == '=' {
				t.Fatalf("GenerateSessionToken() contains forbidden character %q", char)
			}
		}
	}
}

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

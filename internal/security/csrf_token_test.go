package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewCSRFCookie(t *testing.T) {
	cookie, err := NewCSRFCookie("__Host-test_csrf")
	if err != nil {
		t.Fatalf("NewCSRFCookie() error = %v", err)
	}

	if cookie.name != "__Host-test_csrf" {
		t.Fatalf("cookie.name = %q, want %q", cookie.name, "__Host-test_csrf")
	}
	if cookie.path != "/" {
		t.Fatalf("cookie.path = %q, want /", cookie.path)
	}
}

func TestNewCSRFCookieRejectsNonHostPrefix(t *testing.T) {
	if _, err := NewCSRFCookie("tabungan_siswa_csrf"); err != ErrInvalidCSRFCookieName {
		t.Fatalf("NewCSRFCookie() error = %v, want %v", err, ErrInvalidCSRFCookieName)
	}
}

func TestGenerateCSRFToken(t *testing.T) {
	first, err := GenerateCSRFToken()
	if err != nil {
		t.Fatalf("GenerateCSRFToken() error = %v", err)
	}
	second, err := GenerateCSRFToken()
	if err != nil {
		t.Fatalf("GenerateCSRFToken() second error = %v", err)
	}

	if first == "" || second == "" {
		t.Fatal("GenerateCSRFToken() returned empty token")
	}
	if first == second {
		t.Fatal("GenerateCSRFToken() returned duplicate tokens")
	}
	if len(first) < 40 {
		t.Fatalf("GenerateCSRFToken() token length = %d, want at least 40", len(first))
	}
	if strings.ContainsAny(first, "+/=") {
		t.Fatalf("GenerateCSRFToken() token contains non-URL-safe encoding: %q", first)
	}
}

func TestCSRFCookieSet(t *testing.T) {
	recorder := httptest.NewRecorder()
	cookie := NewDefaultCSRFCookie()

	if err := cookie.Set(recorder, "csrf-token"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	setCookie := recorder.Header().Get("Set-Cookie")
	if setCookie == "" {
		t.Fatal("Set() did not write Set-Cookie")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Cookie", setCookie)
	token, err := cookie.Read(req)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if token != "csrf-token" {
		t.Fatalf("Read() token = %q, want csrf-token", token)
	}

	if !strings.Contains(setCookie, "Secure") {
		t.Fatalf("Set-Cookie = %q, missing Secure", setCookie)
	}
	if !strings.Contains(setCookie, "Path=/") {
		t.Fatalf("Set-Cookie = %q, missing Path=/", setCookie)
	}
	if !strings.Contains(setCookie, "SameSite=Lax") {
		t.Fatalf("Set-Cookie = %q, missing SameSite=Lax", setCookie)
	}
	if strings.Contains(setCookie, "HttpOnly") {
		t.Fatalf("Set-Cookie = %q, must not contain HttpOnly", setCookie)
	}
	if strings.Contains(setCookie, "Domain=") {
		t.Fatalf("Set-Cookie = %q, must not contain Domain", setCookie)
	}
}

func TestCSRFCookieSetRejectsEmptyToken(t *testing.T) {
	if err := NewDefaultCSRFCookie().Set(httptest.NewRecorder(), "   "); err != ErrEmptyCSRFToken {
		t.Fatalf("Set() error = %v, want %v", err, ErrEmptyCSRFToken)
	}
}

func TestCSRFCookieReadErrors(t *testing.T) {
	cookie := NewDefaultCSRFCookie()

	t.Run("missing", func(t *testing.T) {
		_, err := cookie.Read(httptest.NewRequest(http.MethodGet, "/", nil))
		if err != ErrCSRFCookieNotFound {
			t.Fatalf("Read() error = %v, want %v", err, ErrCSRFCookieNotFound)
		}
	})

	t.Run("empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  DefaultCSRFCookieName,
			Value: "",
		})

		_, err := cookie.Read(req)
		if err != ErrInvalidCSRFCookie {
			t.Fatalf("Read() error = %v, want %v", err, ErrInvalidCSRFCookie)
		}
	})
}
